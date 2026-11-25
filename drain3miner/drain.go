package drain3miner

import (
	"cmp"
	"container/heap"
	"encoding/json"
	"hash/fnv"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/SigNoz/baseplate.go/errors"
)

// The codes of the errors [Miner.State] and [Miner.LoadState] return.
var (
	CodeStateInvalid      = errors.MustNewCode("state_invalid")
	CodeStateEncodeFailed = errors.MustNewCode("state_encode_failed")
)

const drainStateVersion = 1

// drain is the [ModeStream] miner, following Drain3's Drain class.
//
// Lines of different token counts never meet in Drain, so each token count
// has its own group: a search tree, the clusters in it and a lock. A line
// that fits a pattern as it is, the common case, only reads its group, so
// Process looks under the group's read lock and counts the line with
// atomics; it takes the group's write lock only to create a pattern or
// widen a template. Lines of other token counts go on meanwhile.
//
// What the groups share is in the forest: the groups by token count, the
// last cluster ID, the use stamps and, with MaxClusters, the clusters by
// recency. LoadState builds a new forest and swaps it in.
type drain struct {
	config DrainConfig
	forest atomic.Pointer[forest]
}

// forest is a drain's patterns. Groups maps token counts to groups and is
// copied on write under mu, so a lookup takes no lock.
type forest struct {
	config  *DrainConfig
	mu      sync.Mutex
	groups  atomic.Pointer[map[int]*group]
	lastID  atomic.Int64
	used    atomic.Uint64 // Last use stamp handed out, for MaxClusters.
	evictMu sync.Mutex    // EvictMu guards stamped, live, swept and every group's dropped.
	stamped stampHeap     // Every kept cluster, by the use stamp it had when pushed or last fixed; only with MaxClusters.
	live    int           // Kept clusters; only with MaxClusters.
	swept   []*group      // Groups with dropped clusters, to sweep once the dropping call lets go of its locks.
	pending atomic.Int64  // Len of swept, read without evictMu.
}

// group holds the patterns of one token count. Mu guards root, clusters
// and version. A cluster dropped for MaxClusters is marked by the forest
// without the group's lock, and stays in clusters, ignored, until the call
// that dropped it has let go of its locks and sweeps the group.
type group struct {
	forest     *forest
	tokenCount int
	mu         sync.RWMutex
	root       *node              // Nil until the group's first cluster.
	clusters   map[int64]*cluster // By cluster ID.
	version    uint64             // Bumped whenever a pattern is created or widened.
	dropped    []int64            // IDs of clusters marked dropped since the last sweep, under the forest's evictMu.
}

// node is an inner node of the search tree. Its children are keyed by
// token, with ParamString as the key of the child that any token reaches.
type node struct {
	children   map[string]*node
	order      []string // Children's keys in the order they were added.
	clusterIDs []int64  // May hold IDs of dropped clusters until the next insert here.
}

// cluster is a pattern. Its id, counting up from 1 in creation order, keys
// the tree and the state; the pattern's ID is the hash of its template.
type cluster struct {
	id       int64
	tokens   []string
	template string
	hash     uint64
	count    atomic.Int64
	lastUsed atomic.Uint64 // Use stamp, higher is more recent, or droppedStamp; set only with MaxClusters.
}

// droppedStamp is the use stamp of a cluster MaxClusters dropped. Using
// and dropping a cluster both swap its stamp, so of a use and a drop that
// race, exactly one takes effect.
const droppedStamp = math.MaxUint64

// drainState is the JSON form of a drain's state.
type drainState struct {
	Version     int            `json:"version"`
	Depth       int            `json:"depth"` // Depth and ParamString shape the tree, so a miner loads only its own.
	ParamString string         `json:"param_string"`
	LastID      int64          `json:"last_id"`
	Clusters    []clusterState `json:"clusters"` // Least recently used first.
	Root        []rootState    `json:"root"`
}

type clusterState struct {
	ID     int64    `json:"id"`
	Tokens []string `json:"tokens"`
	Count  int64    `json:"count"`
}

type rootState struct {
	TokenCount int       `json:"token_count"`
	Node       nodeState `json:"node"`
}

type nodeState struct {
	Children   []childState `json:"children,omitempty"` // In the order they were added.
	ClusterIDs []int64      `json:"cluster_ids,omitempty"`
}

type childState struct {
	Key  string    `json:"key"`
	Node nodeState `json:"node"`
}

// batchLine is one distinct line of a ProcessBatch call. Cluster is the
// pattern it fits as it is, valid while its group's version is still
// version and the cluster isn't dropped, which another group's new
// pattern can do; nil means it must be searched again.
type batchLine struct {
	tokens  []string
	group   *group
	cluster *cluster
	version uint64
}

// batchCount gathers the lines of a batch that a cluster counts.
type batchCount struct {
	cluster *cluster
	lines   int64
	base    int64 // Count before the batch.
	last    int   // Index of the batch's last line in the cluster.
}

func newDrain(config DrainConfig) *drain {
	config.ExtraDelimiters = slices.Clone(config.ExtraDelimiters)

	d := &drain{config: config}
	d.forest.Store(newForest(&d.config))

	return d
}

// Process puts line into the best matching pattern or a new one. It first
// looks under its group's read lock and, when the line fits a pattern
// without widening it, only counts it there. Otherwise it searches again
// under the group's write lock, since another call may have changed the
// group in between.
func (d *drain) Process(line string) (Pattern, *Change) {
	tokens := d.tokenize(line)
	g := d.forest.Load().group(len(tokens))

	g.mu.RLock()
	if c := g.treeSearch(tokens, d.config.SimThreshold, false); c != nil && g.forest.fits(c, tokens) && g.forest.touch(c) {
		c.count.Add(1)
		pattern := c.pattern()
		g.mu.RUnlock()

		return pattern, nil
	}
	g.mu.RUnlock()

	g.mu.Lock()
	c, change := g.process(tokens)
	pattern := c.pattern()
	g.mu.Unlock()

	g.forest.sweep()

	return pattern, change
}

// ProcessBatch processes lines as Process would one after another, and
// returns each line's pattern and change at the same index. Each distinct
// line is first searched under the read locks of the groups the lines fall
// in, taken in token-count order; when every line fits a pattern as it is,
// the counts are added there. Otherwise the batch runs again in order
// under those groups' write locks, searching only the lines whose earlier
// result no longer holds. A line's Count is the pattern's count as Process
// would return it.
func (d *drain) ProcessBatch(lines []string) ([]Pattern, []*Change) {
	if len(lines) == 0 {
		return nil, nil
	}

	f := d.forest.Load()
	index := make(map[string]int, len(lines))
	lineOf := make([]int, len(lines))
	distinct := make([]batchLine, 0)
	groups := make([]*group, 0)

	for i, line := range lines {
		j, ok := index[line]
		if !ok {
			j = len(distinct)
			index[line] = j
			tokens := d.tokenize(line)
			g := f.group(len(tokens))
			distinct = append(distinct, batchLine{
				tokens: tokens,
				group:  g,
			})

			if !slices.Contains(groups, g) {
				groups = append(groups, g)
			}
		}

		lineOf[i] = j
	}

	slices.SortFunc(groups, func(a, b *group) int {
		return cmp.Compare(a.tokenCount, b.tokenCount)
	})

	for _, g := range groups {
		g.mu.RLock()
	}

	patterns, ok := f.countBatch(lineOf, distinct)

	for _, g := range groups {
		g.mu.RUnlock()
	}

	if ok {
		return patterns, make([]*Change, len(lines))
	}

	for _, g := range groups {
		g.mu.Lock()
	}

	defer f.sweep()
	defer func() {
		for _, g := range groups {
			g.mu.Unlock()
		}
	}()

	patterns = make([]Pattern, len(lines))
	changes := make([]*Change, len(lines))

	for i, j := range lineOf {
		entry := &distinct[j]
		if entry.cluster != nil && entry.version == entry.group.version && f.touch(entry.cluster) {
			entry.cluster.count.Add(1)
			patterns[i] = entry.cluster.pattern()

			continue
		}

		c, change := entry.group.process(entry.tokens)
		patterns[i] = c.pattern()
		changes[i] = change
		entry.cluster = nil

		if change == nil {
			entry.cluster = c
			entry.version = entry.group.version
		}
	}

	return patterns, changes
}

// Match finds the pattern line fully matches: first along the search tree,
// then, when the tree finds none, among every pattern of the line's token
// count. Of equally good patterns it takes the one with the most
// parameters, as Drain3 does.
func (d *drain) Match(line string) (Pattern, bool) {
	tokens := d.tokenize(line)

	g := d.forest.Load().lookup(len(tokens))
	if g == nil {
		return Pattern{}, false
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	c := g.treeSearch(tokens, 1, true)
	if c == nil && g.root != nil {
		c = g.bestMatch(g.root.collect(nil), tokens, 1, true)
	}

	if c == nil {
		return Pattern{}, false
	}

	return c.pattern(), true
}

// Patterns lists the kept patterns in the order they were created.
func (d *drain) Patterns() []Pattern {
	f := d.forest.Load()
	groups := f.lockAll()
	defer f.unlockAll(groups)

	clusters := f.kept(groups)
	slices.SortFunc(clusters, func(a, b *cluster) int {
		return cmp.Compare(a.id, b.id)
	})

	patterns := make([]Pattern, 0, len(clusters))
	for _, c := range clusters {
		patterns = append(patterns, c.pattern())
	}

	return patterns
}

// State encodes the clusters, least recently used first, the search trees
// and the last ID as JSON.
func (d *drain) State() ([]byte, error) {
	f := d.forest.Load()
	groups := f.lockAll()
	defer f.unlockAll(groups)

	clusters := f.kept(groups)
	slices.SortFunc(clusters, func(a, b *cluster) int {
		return cmp.Or(cmp.Compare(a.lastUsed.Load(), b.lastUsed.Load()), cmp.Compare(a.id, b.id))
	})

	state := drainState{
		Version:     drainStateVersion,
		Depth:       d.config.Depth,
		ParamString: d.config.ParamString,
		LastID:      f.lastID.Load(),
		Clusters:    make([]clusterState, 0, len(clusters)),
	}

	for _, c := range clusters {
		state.Clusters = append(state.Clusters, clusterState{
			ID:     c.id,
			Tokens: c.tokens,
			Count:  c.count.Load(),
		})
	}

	for _, g := range groups {
		if g.root == nil {
			continue
		}

		state.Root = append(state.Root, rootState{
			TokenCount: g.tokenCount,
			Node:       g.root.state(),
		})
	}

	encoded, err := json.Marshal(state)
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInternal, CodeStateEncodeFailed, "encode drain state")
	}

	return encoded, nil
}

// LoadState decodes state from [drain.State] into a new forest and swaps it
// in. A call that started on the old forest finishes there, as if it ran
// before the load.
func (d *drain) LoadState(state []byte) error {
	var decoded drainState
	if err := json.Unmarshal(state, &decoded); err != nil {
		return errors.Wrap(err, errors.TypeInvalidInput, CodeStateInvalid, "drain state doesn't parse")
	}

	switch {
	case decoded.Version != drainStateVersion:
		return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state version %d is invalid, expected %d", decoded.Version, drainStateVersion)
	case decoded.Depth != d.config.Depth:
		return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state depth %d is invalid, expected the miner's depth %d", decoded.Depth, d.config.Depth)
	case decoded.ParamString != d.config.ParamString:
		return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state param string %q is invalid, expected the miner's param string %q", decoded.ParamString, d.config.ParamString)
	}

	loaded := newForest(&d.config)
	loaded.lastID.Store(decoded.LastID)

	byID := make(map[int64]*cluster, len(decoded.Clusters))
	for _, c := range decoded.Clusters {
		if c.ID < 1 || c.ID > decoded.LastID || byID[c.ID] != nil {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected a unique ID from 1 to the last ID %d", c.ID, decoded.LastID)
		}

		if c.Count < 1 || slices.ContainsFunc(c.Tokens, func(token string) bool {
			return token == "" || strings.ContainsFunc(token, isSpace)
		}) {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected a positive count and tokens without whitespace", c.ID)
		}

		kept := newCluster(c.ID, c.Tokens, c.Count)
		byID[c.ID] = kept
		loaded.group(len(c.Tokens)).insert(kept)
	}

	seen := make(map[int]bool, len(decoded.Root))
	for _, root := range decoded.Root {
		if root.TokenCount < 0 || seen[root.TokenCount] {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state token count %d is invalid, expected a unique count of 0 or more", root.TokenCount)
		}

		seen[root.TokenCount] = true

		n, err := newNode(root.Node)
		if err != nil {
			return err
		}

		for _, id := range n.collect(nil) {
			if c := byID[id]; c != nil && len(c.tokens) != root.TokenCount {
				return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected %d tokens to match its place in the tree", id, root.TokenCount)
			}
		}

		loaded.group(root.TokenCount).root = n
	}

	loaded.sweep()
	d.forest.Store(loaded)

	return nil
}

// tokenize splits line into tokens. Invalid UTF-8 becomes U+FFFD first, as
// it would in the JSON state, so a template and its ID survive a restore.
func (d *drain) tokenize(line string) []string {
	if !utf8.ValidString(line) {
		line = strings.ToValidUTF8(line, "\uFFFD")
	}

	for _, delimiter := range d.config.ExtraDelimiters {
		line = strings.ReplaceAll(line, delimiter, " ")
	}

	return strings.FieldsFunc(line, isSpace)
}

// newForest returns an empty forest for config.
func newForest(config *DrainConfig) *forest {
	f := &forest{config: config}
	groups := map[int]*group{}
	f.groups.Store(&groups)

	return f
}

// group returns the group for tokenCount, adding it when there is none.
func (f *forest) group(tokenCount int) *group {
	if g := f.lookup(tokenCount); g != nil {
		return g
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	groups := *f.groups.Load()
	if g := groups[tokenCount]; g != nil {
		return g
	}

	g := &group{
		forest:     f,
		tokenCount: tokenCount,
		clusters:   map[int64]*cluster{},
	}

	copied := maps.Clone(groups)
	copied[tokenCount] = g
	f.groups.Store(&copied)

	return g
}

// lookup returns the group for tokenCount, or nil.
func (f *forest) lookup(tokenCount int) *group {
	return (*f.groups.Load())[tokenCount]
}

// lockAll read-locks every group in token-count order and returns them,
// with evictMu held so no cluster is dropped meanwhile. A group added
// while it locks may hold a cluster that dropped one of the locked groups'
// clusters, so it then starts over. A group's writer takes evictMu under
// the group's lock, so lockAll takes it last.
func (f *forest) lockAll() []*group {
	for {
		byCount := f.groups.Load()
		groups := slices.SortedFunc(maps.Values(*byCount), func(a, b *group) int {
			return cmp.Compare(a.tokenCount, b.tokenCount)
		})

		for _, g := range groups {
			g.mu.RLock()
		}

		f.evictMu.Lock()

		if f.groups.Load() == byCount {
			return groups
		}

		f.unlockAll(groups)
	}
}

// unlockAll lets go of the locks [forest.lockAll] took.
func (f *forest) unlockAll(groups []*group) {
	f.evictMu.Unlock()

	for _, g := range groups {
		g.mu.RUnlock()
	}
}

// kept returns the clusters of groups that aren't dropped. The groups must
// be locked.
func (f *forest) kept(groups []*group) []*cluster {
	var clusters []*cluster
	for _, g := range groups {
		for _, c := range g.clusters {
			if !c.isDropped() {
				clusters = append(clusters, c)
			}
		}
	}

	return clusters
}

// countBatch finds, under the groups' read locks, the cluster each
// distinct line fits as it is and counts the batch's lines there, touching
// the clusters in the order of their last line. It returns false, counting
// nothing, when a line fits none; the lines it found keep their cluster for
// the write-locked pass.
func (f *forest) countBatch(lineOf []int, distinct []batchLine) ([]Pattern, bool) {
	for j := range distinct {
		entry := &distinct[j]

		c := entry.group.treeSearch(entry.tokens, f.config.SimThreshold, false)
		if c == nil || !f.fits(c, entry.tokens) {
			return nil, false
		}

		entry.cluster = c
		entry.version = entry.group.version
	}

	counts := make(map[*cluster]*batchCount)
	ordered := make([]*batchCount, 0)

	for i, j := range lineOf {
		c := distinct[j].cluster

		count := counts[c]
		if count == nil {
			count = &batchCount{cluster: c}
			counts[c] = count
			ordered = append(ordered, count)
		}

		count.lines++
		count.last = i
	}

	slices.SortFunc(ordered, func(a, b *batchCount) int {
		return cmp.Compare(a.last, b.last)
	})

	for _, count := range ordered {
		if !f.touch(count.cluster) {
			return nil, false
		}
	}

	for _, count := range ordered {
		count.base = count.cluster.count.Add(count.lines) - count.lines
	}

	patterns := make([]Pattern, len(lineOf))
	for i, j := range lineOf {
		count := counts[distinct[j].cluster]
		count.base++
		patterns[i] = Pattern{
			ID:       count.cluster.hash,
			Template: count.cluster.template,
			Count:    count.base,
		}
	}

	return patterns, true
}

// keep makes room for a new cluster under MaxClusters, marking the least
// recently used clusters dropped, and starts tracking c's use.
func (f *forest) keep(c *cluster) {
	// A cluster not yet kept can't be dropped, so the touch takes.
	_ = f.touch(c)

	if f.config.MaxClusters == 0 {
		return
	}

	f.evictMu.Lock()
	defer f.evictMu.Unlock()

	for f.live >= f.config.MaxClusters {
		f.dropLeastRecent()
	}

	heap.Push(&f.stamped, stampedCluster{
		cluster: c,
		stamp:   c.lastUsed.Load(),
	})
	f.live++
}

// dropLeastRecent marks the least recently used cluster dropped, under
// evictMu. Touches don't reorder the heap, so a top cluster used since it
// was stamped is restamped and sifted down first; the first top that
// wasn't used since is older than every other cluster.
func (f *forest) dropLeastRecent() {
	for {
		top := &f.stamped[0]
		if used := top.cluster.lastUsed.Load(); used != top.stamp {
			top.stamp = used
			heap.Fix(&f.stamped, 0)

			continue
		}

		// A use since the load above wins, and the top is looked at again.
		if !top.cluster.lastUsed.CompareAndSwap(top.stamp, droppedStamp) {
			continue
		}

		dropped, _ := heap.Pop(&f.stamped).(stampedCluster)
		f.live--

		g := f.lookup(len(dropped.cluster.tokens))
		if len(g.dropped) == 0 {
			f.swept = append(f.swept, g)
			f.pending.Add(1)
		}

		g.dropped = append(g.dropped, dropped.cluster.id)

		return
	}
}

// touch marks c as the most recently used cluster, and reports false,
// changing nothing, when c was dropped. A caller counts a line in c only
// after touch reports true. It needs no lock.
func (f *forest) touch(c *cluster) bool {
	if f.config.MaxClusters == 0 {
		return true
	}

	// A concurrent touch may have stored a newer stamp since this one was
	// taken; keep the newest.
	stamp := f.used.Add(1)
	for {
		last := c.lastUsed.Load()
		if last == droppedStamp {
			return false
		}

		if last >= stamp || c.lastUsed.CompareAndSwap(last, stamp) {
			return true
		}
	}
}

// sweep deletes the clusters dropped for MaxClusters from their groups.
// A call that may drop clusters runs it after letting go of its locks.
func (f *forest) sweep() {
	if f.pending.Load() == 0 {
		return
	}

	f.evictMu.Lock()
	groups := f.swept
	f.swept = nil
	f.pending.Store(0)
	f.evictMu.Unlock()

	for _, g := range groups {
		g.mu.Lock()
		f.evictMu.Lock()
		ids := g.dropped
		g.dropped = nil
		f.evictMu.Unlock()

		for _, id := range ids {
			delete(g.clusters, id)
		}
		g.mu.Unlock()
	}
}

// fits reports whether tokens fit c's template as it is: each token equals
// the template's or the template has a parameter there.
func (f *forest) fits(c *cluster, tokens []string) bool {
	for i, token := range tokens {
		if token != c.tokens[i] && c.tokens[i] != f.config.ParamString {
			return false
		}
	}

	return true
}

// similarity returns the share of template positions whose token equals the
// line's, counting parameters as equal when includeParams is set, and the
// number of parameters.
func (f *forest) similarity(template, tokens []string, includeParams bool) (float64, int) {
	if len(template) == 0 {
		return 1, 0
	}

	equal := 0
	params := 0

	for i, token := range template {
		if token == f.config.ParamString {
			params++

			continue
		}

		if token == tokens[i] {
			equal++
		}
	}

	if includeParams {
		equal += params
	}

	return float64(equal) / float64(len(template)), params
}

// process puts tokens into the best matching cluster or a new one, under
// the group's write lock, and bumps the version when it creates or widens
// one.
func (g *group) process(tokens []string) (*cluster, *Change) {
	f := g.forest

	// Another group's new cluster may drop the one found before it is
	// used; the search then runs again as if the drop came first.
	c := g.treeSearch(tokens, f.config.SimThreshold, false)
	for c != nil && !f.touch(c) {
		c = g.treeSearch(tokens, f.config.SimThreshold, false)
	}

	if c == nil {
		// The cluster widens its tokens in place, and a batch reuses a
		// line's tokens, so the cluster gets its own copy.
		c = newCluster(f.lastID.Add(1), slices.Clone(tokens), 1)
		g.insert(c)
		g.addToTree(c)
		g.version++

		return c, &Change{To: c.hash}
	}

	changed := false
	for i, token := range tokens {
		if token != c.tokens[i] && c.tokens[i] != f.config.ParamString {
			c.tokens[i] = f.config.ParamString
			changed = true
		}
	}

	var change *Change
	if changed {
		from := c.hash
		c.template = strings.Join(c.tokens, " ")
		c.hash = hashTemplate(c.template)
		g.version++
		change = &Change{
			From: from,
			To:   c.hash,
		}
	}

	c.count.Add(1)

	return c, change
}

// insert keeps a new cluster in the group, under its write lock, after
// making room under MaxClusters. A dropped cluster's ID stays in its leaf
// until the next insert there.
func (g *group) insert(c *cluster) {
	g.forest.keep(c)
	g.clusters[c.id] = c
}

// cluster returns the kept cluster for id, or nil when it was dropped.
func (g *group) cluster(id int64) *cluster {
	c := g.clusters[id]
	if c == nil || c.isDropped() {
		return nil
	}

	return c
}

// treeSearch follows tokens down the group's tree, taking the parameter's
// child where no child has the token, and returns the best cluster at the
// leaf that reaches threshold, or nil.
func (g *group) treeSearch(tokens []string, threshold float64, includeParams bool) *cluster {
	current := g.root
	if current == nil {
		return nil
	}

	if len(tokens) == 0 {
		if len(current.clusterIDs) == 0 {
			return nil
		}

		return g.cluster(current.clusterIDs[0])
	}

	depth := 1
	for _, token := range tokens {
		if depth >= g.forest.config.Depth-2 || depth == len(tokens) {
			break
		}

		next := current.children[token]
		if next == nil {
			next = current.children[g.forest.config.ParamString]
		}

		if next == nil {
			return nil
		}

		current = next
		depth++
	}

	return g.bestMatch(current.clusterIDs, tokens, threshold, includeParams)
}

// bestMatch returns the cluster with the highest similarity to tokens, ties
// going to the one with more parameters and then to the first, or nil when
// the highest is below threshold.
func (g *group) bestMatch(ids []int64, tokens []string, threshold float64, includeParams bool) *cluster {
	var best *cluster

	bestSimilarity := -1.0
	bestParams := -1

	for _, id := range ids {
		c := g.cluster(id)
		if c == nil {
			continue
		}

		similarity, params := g.forest.similarity(c.tokens, tokens, includeParams)
		if similarity > bestSimilarity || (similarity == bestSimilarity && params > bestParams) {
			best = c
			bestSimilarity = similarity
			bestParams = params
		}
	}

	if bestSimilarity < threshold {
		return nil
	}

	return best
}

// addToTree adds a new cluster to the leaf its tokens lead to, creating the
// nodes on the way. A token with a digit, or one that would exceed
// MaxChildren, takes the parameter's child.
func (g *group) addToTree(c *cluster) {
	tokenCount := len(c.tokens)

	if g.root == nil {
		g.root = &node{}
	}

	current := g.root

	if tokenCount == 0 {
		current.clusterIDs = []int64{c.id}

		return
	}

	config := g.forest.config
	param := config.ParamString
	depth := 1

	for _, token := range c.tokens {
		if depth >= config.Depth-2 || depth >= tokenCount {
			kept := current.clusterIDs[:0]
			for _, id := range current.clusterIDs {
				if g.cluster(id) != nil {
					kept = append(kept, id)
				}
			}

			current.clusterIDs = append(kept, c.id)

			break
		}

		if next := current.children[token]; next != nil {
			current = next
			depth++

			continue
		}

		_, hasParam := current.children[param]
		children := len(current.children)

		switch {
		case !config.KeepNumericTokens && strings.ContainsFunc(token, unicode.IsDigit):
			current = current.child(param)
		case hasParam && children < config.MaxChildren:
			current = current.child(token)
		case hasParam:
			current = current.children[param]
		case children+1 < config.MaxChildren:
			current = current.child(token)
		default:
			current = current.child(param)
		}

		depth++
	}
}

// child returns the child for key, adding it when there is none.
func (n *node) child(key string) *node {
	if next := n.children[key]; next != nil {
		return next
	}

	if n.children == nil {
		n.children = map[string]*node{}
	}

	next := &node{}
	n.children[key] = next
	n.order = append(n.order, key)

	return next
}

// collect appends the cluster IDs of n and its descendants to ids, the
// node's own first and then its children's in the order they were added.
func (n *node) collect(ids []int64) []int64 {
	ids = append(ids, n.clusterIDs...)
	for _, key := range n.order {
		ids = n.children[key].collect(ids)
	}

	return ids
}

func (n *node) state() nodeState {
	state := nodeState{ClusterIDs: n.clusterIDs}
	for _, key := range n.order {
		state.Children = append(state.Children, childState{
			Key:  key,
			Node: n.children[key].state(),
		})
	}

	return state
}

// isDropped reports whether MaxClusters dropped c.
func (c *cluster) isDropped() bool {
	return c.lastUsed.Load() == droppedStamp
}

// newCluster returns a cluster for tokens, with its template and hash.
func newCluster(id int64, tokens []string, count int64) *cluster {
	template := strings.Join(tokens, " ")

	c := &cluster{
		id:       id,
		tokens:   tokens,
		template: template,
		hash:     hashTemplate(template),
	}
	c.count.Store(count)

	return c
}

// hashTemplate returns a template's pattern ID: the 64-bit FNV-1a hash of
// its bytes, the same in every process and version.
func hashTemplate(template string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(template))

	return h.Sum64()
}

func (c *cluster) pattern() Pattern {
	return Pattern{
		ID:       c.hash,
		Template: c.template,
		Count:    c.count.Load(),
	}
}

// isSpace reports whether r separates tokens: Unicode whitespace and the
// information separators U+001C to U+001F, as in Python's str.split.
func isSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// newNode builds a tree node and its descendants from their state. IDs of
// clusters the state doesn't hold are kept, as dropped clusters are.
func newNode(state nodeState) (*node, error) {
	n := &node{clusterIDs: state.ClusterIDs}
	for _, child := range state.Children {
		if _, ok := n.children[child.Key]; ok || child.Key == "" {
			return nil, errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state tree key %q is invalid, expected a unique token", child.Key)
		}

		next, err := newNode(child.Node)
		if err != nil {
			return nil, err
		}

		if n.children == nil {
			n.children = map[string]*node{}
		}

		n.children[child.Key] = next
		n.order = append(n.order, child.Key)
	}

	return n, nil
}

// stampedCluster is a cluster with the use stamp it is ordered by in a
// [stampHeap].
type stampedCluster struct {
	cluster *cluster
	stamp   uint64
}

// stampHeap is a min-heap of clusters by stamp, for [container/heap].
type stampHeap []stampedCluster

func (h stampHeap) Len() int           { return len(h) }
func (h stampHeap) Less(i, j int) bool { return h[i].stamp < h[j].stamp }
func (h stampHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *stampHeap) Push(x any) {
	if c, ok := x.(stampedCluster); ok {
		*h = append(*h, c)
	}
}

func (h *stampHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]

	return last
}
