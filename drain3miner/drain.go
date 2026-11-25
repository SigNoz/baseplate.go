package drain3miner

import (
	"cmp"
	"container/heap"
	"encoding/json"
	"hash/fnv"
	"maps"
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
// The mu lock guards the tree and the clusters. A line that fits a pattern
// as it is, the common case, only reads them, so Process looks under the
// read lock and counts the line with atomics; it takes the write lock only
// to create a pattern or widen a template. LoadState also takes the write
// lock, and the other methods the read lock.
type drain struct {
	mu       sync.RWMutex
	config   DrainConfig
	root     map[int]*node      // First layer, keyed by token count.
	clusters map[int64]*cluster // Patterns kept, by cluster ID.
	lastID   int64
	version  uint64        // Bumped whenever a pattern is created or widened, or state is loaded.
	used     atomic.Uint64 // Last use stamp handed out, for MaxClusters.
	stamped  stampHeap     // Every cluster, by the use stamp it had when pushed or last fixed; only with MaxClusters.
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
	lastUsed atomic.Uint64 // Use stamp, higher is more recent; set only with MaxClusters.
}

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
// pattern it fits as it is, valid while the drain's version is still
// version; nil means it must be searched again.
type batchLine struct {
	tokens  []string
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

	return &drain{
		config:   config,
		root:     map[int]*node{},
		clusters: map[int64]*cluster{},
	}
}

// Process puts line into the best matching pattern or a new one. It first
// looks under the read lock and, when the line fits a pattern without
// widening it, only counts it there. Otherwise it searches again under the
// write lock, since another call may have changed the tree in between.
func (d *drain) Process(line string) (Pattern, *Change) {
	tokens := d.tokenize(line)

	d.mu.RLock()
	if c := d.treeSearch(tokens, d.config.SimThreshold, false); c != nil && d.fits(c, tokens) {
		c.count.Add(1)
		d.touch(c)
		pattern := c.pattern()
		d.mu.RUnlock()

		return pattern, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	c, change := d.processLocked(tokens)

	return c.pattern(), change
}

// ProcessBatch processes lines as Process would one after another, and
// returns each line's pattern and change at the same index. Each distinct
// line is first searched under the read lock; when every line fits a
// pattern as it is, the counts are added there. Otherwise the batch runs
// again in order under the write lock, searching only the lines whose
// earlier result no longer holds. A line's Count is the pattern's count
// as Process would return it.
func (d *drain) ProcessBatch(lines []string) ([]Pattern, []*Change) {
	if len(lines) == 0 {
		return nil, nil
	}

	index := make(map[string]int, len(lines))
	lineOf := make([]int, len(lines))
	distinct := make([]batchLine, 0)

	for i, line := range lines {
		j, ok := index[line]
		if !ok {
			j = len(distinct)
			index[line] = j
			distinct = append(distinct, batchLine{tokens: d.tokenize(line)})
		}

		lineOf[i] = j
	}

	d.mu.RLock()
	if patterns, ok := d.countBatch(lineOf, distinct); ok {
		d.mu.RUnlock()

		return patterns, make([]*Change, len(lines))
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	patterns := make([]Pattern, len(lines))
	changes := make([]*Change, len(lines))

	for i, j := range lineOf {
		entry := &distinct[j]
		if entry.cluster != nil && entry.version == d.version {
			entry.cluster.count.Add(1)
			d.touch(entry.cluster)
			patterns[i] = entry.cluster.pattern()

			continue
		}

		c, change := d.processLocked(entry.tokens)
		patterns[i] = c.pattern()
		changes[i] = change
		entry.cluster = nil

		if change == nil {
			entry.cluster = c
			entry.version = d.version
		}
	}

	return patterns, changes
}

// Match finds the pattern line fully matches: first along the search tree,
// then, when the tree finds none, among every pattern of the line's token
// count. Of equally good patterns it takes the one with the most
// parameters, as Drain3 does.
func (d *drain) Match(line string) (Pattern, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tokens := d.tokenize(line)

	c := d.treeSearch(tokens, 1, true)
	if c == nil {
		var ids []int64
		if first := d.root[len(tokens)]; first != nil {
			ids = first.collect(ids)
		}

		c = d.bestMatch(ids, tokens, 1, true)
	}

	if c == nil {
		return Pattern{}, false
	}

	return c.pattern(), true
}

// Patterns lists the kept patterns in the order they were created.
func (d *drain) Patterns() []Pattern {
	d.mu.RLock()
	defer d.mu.RUnlock()

	clusters := slices.SortedFunc(maps.Values(d.clusters), func(a, b *cluster) int {
		return cmp.Compare(a.id, b.id)
	})

	patterns := make([]Pattern, 0, len(clusters))
	for _, c := range clusters {
		patterns = append(patterns, c.pattern())
	}

	return patterns
}

// State encodes the clusters, least recently used first, the search tree
// and the last ID as JSON.
func (d *drain) State() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	state := drainState{
		Version:     drainStateVersion,
		Depth:       d.config.Depth,
		ParamString: d.config.ParamString,
		LastID:      d.lastID,
		Clusters:    make([]clusterState, 0, len(d.clusters)),
	}

	for _, c := range d.leastRecentFirst() {
		state.Clusters = append(state.Clusters, clusterState{
			ID:     c.id,
			Tokens: c.tokens,
			Count:  c.count.Load(),
		})
	}

	counts := slices.Sorted(maps.Keys(d.root))
	for _, count := range counts {
		state.Root = append(state.Root, rootState{
			TokenCount: count,
			Node:       d.root[count].state(),
		})
	}

	encoded, err := json.Marshal(state)
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInternal, CodeStateEncodeFailed, "encode drain state")
	}

	return encoded, nil
}

// LoadState decodes state from [drain.State] and replaces the clusters,
// the tree and the last ID with it.
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

	loaded := newDrain(d.config)
	loaded.lastID = decoded.LastID

	for _, c := range decoded.Clusters {
		if c.ID < 1 || c.ID > decoded.LastID || loaded.clusters[c.ID] != nil {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected a unique ID from 1 to the last ID %d", c.ID, decoded.LastID)
		}

		if c.Count < 1 || slices.ContainsFunc(c.Tokens, func(token string) bool {
			return token == "" || strings.ContainsFunc(token, isSpace)
		}) {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected a positive count and tokens without whitespace", c.ID)
		}

		loaded.insert(newCluster(c.ID, c.Tokens, c.Count))
	}

	for _, root := range decoded.Root {
		if root.TokenCount < 0 || loaded.root[root.TokenCount] != nil {
			return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state token count %d is invalid, expected a unique count of 0 or more", root.TokenCount)
		}

		n, err := newNode(root.Node)
		if err != nil {
			return err
		}

		for _, id := range n.collect(nil) {
			if c := loaded.clusters[id]; c != nil && len(c.tokens) != root.TokenCount {
				return errors.Newf(errors.TypeInvalidInput, CodeStateInvalid, "drain state cluster %d is invalid, expected %d tokens to match its place in the tree", id, root.TokenCount)
			}
		}

		loaded.root[root.TokenCount] = n
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.root = loaded.root
	d.clusters = loaded.clusters
	d.version++
	d.lastID = loaded.lastID
	d.used.Store(loaded.used.Load())
	d.stamped = loaded.stamped

	return nil
}

// processLocked puts tokens into the best matching cluster or a new one,
// under the write lock, and bumps the version when it creates or widens
// one.
func (d *drain) processLocked(tokens []string) (*cluster, *Change) {
	c := d.treeSearch(tokens, d.config.SimThreshold, false)
	if c == nil {
		// The cluster widens its tokens in place, and a batch reuses a
		// line's tokens, so the cluster gets its own copy.
		d.lastID++
		c = newCluster(d.lastID, slices.Clone(tokens), 1)
		d.insert(c)
		d.addToTree(c)
		d.version++

		return c, &Change{To: c.hash}
	}

	changed := false
	for i, token := range tokens {
		if token != c.tokens[i] && c.tokens[i] != d.config.ParamString {
			c.tokens[i] = d.config.ParamString
			changed = true
		}
	}

	var change *Change
	if changed {
		from := c.hash
		c.template = strings.Join(c.tokens, " ")
		c.hash = hashTemplate(c.template)
		d.version++
		change = &Change{
			From: from,
			To:   c.hash,
		}
	}

	c.count.Add(1)
	d.touch(c)

	return c, change
}

// countBatch finds, under the read lock, the cluster each distinct line
// fits as it is and counts the batch's lines there, touching the clusters
// in the order of their last line. It returns false, counting nothing,
// when a line fits none; the lines it found keep their cluster for the
// write-locked pass.
func (d *drain) countBatch(lineOf []int, distinct []batchLine) ([]Pattern, bool) {
	for j := range distinct {
		entry := &distinct[j]

		c := d.treeSearch(entry.tokens, d.config.SimThreshold, false)
		if c == nil || !d.fits(c, entry.tokens) {
			return nil, false
		}

		entry.cluster = c
		entry.version = d.version
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
		count.base = count.cluster.count.Add(count.lines) - count.lines
		d.touch(count.cluster)
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

// treeSearch follows tokens down the tree, taking the parameter's child
// where no child has the token, and returns the best cluster at the leaf
// that reaches threshold, or nil.
func (d *drain) treeSearch(tokens []string, threshold float64, includeParams bool) *cluster {
	current := d.root[len(tokens)]
	if current == nil {
		return nil
	}

	if len(tokens) == 0 {
		if len(current.clusterIDs) == 0 {
			return nil
		}

		return d.clusters[current.clusterIDs[0]]
	}

	depth := 1
	for _, token := range tokens {
		if depth >= d.config.Depth-2 || depth == len(tokens) {
			break
		}

		next := current.children[token]
		if next == nil {
			next = current.children[d.config.ParamString]
		}

		if next == nil {
			return nil
		}

		current = next
		depth++
	}

	return d.bestMatch(current.clusterIDs, tokens, threshold, includeParams)
}

// bestMatch returns the cluster with the highest similarity to tokens, ties
// going to the one with more parameters and then to the first, or nil when
// the highest is below threshold.
func (d *drain) bestMatch(ids []int64, tokens []string, threshold float64, includeParams bool) *cluster {
	var best *cluster

	bestSimilarity := -1.0
	bestParams := -1

	for _, id := range ids {
		c := d.clusters[id]
		if c == nil {
			continue
		}

		similarity, params := d.similarity(c.tokens, tokens, includeParams)
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

// similarity returns the share of template positions whose token equals the
// line's, counting parameters as equal when includeParams is set, and the
// number of parameters.
func (d *drain) similarity(template, tokens []string, includeParams bool) (float64, int) {
	if len(template) == 0 {
		return 1, 0
	}

	equal := 0
	params := 0

	for i, token := range template {
		if token == d.config.ParamString {
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

// addToTree adds a new cluster to the leaf its tokens lead to, creating the
// nodes on the way. A token with a digit, or one that would exceed
// MaxChildren, takes the parameter's child.
func (d *drain) addToTree(c *cluster) {
	tokenCount := len(c.tokens)

	current := d.root[tokenCount]
	if current == nil {
		current = &node{}
		d.root[tokenCount] = current
	}

	if tokenCount == 0 {
		current.clusterIDs = []int64{c.id}

		return
	}

	param := d.config.ParamString
	depth := 1

	for _, token := range c.tokens {
		if depth >= d.config.Depth-2 || depth >= tokenCount {
			kept := current.clusterIDs[:0]
			for _, id := range current.clusterIDs {
				if d.clusters[id] != nil {
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
		case !d.config.KeepNumericTokens && strings.ContainsFunc(token, unicode.IsDigit):
			current = current.child(param)
		case hasParam && children < d.config.MaxChildren:
			current = current.child(token)
		case hasParam:
			current = current.children[param]
		case children+1 < d.config.MaxChildren:
			current = current.child(token)
		default:
			current = current.child(param)
		}

		depth++
	}
}

// insert keeps a new cluster, dropping the least recently used one first
// when MaxClusters is reached. A dropped cluster stays in its leaf's IDs
// until the next insert there.
func (d *drain) insert(c *cluster) {
	d.touch(c)

	if d.config.MaxClusters > 0 {
		for len(d.clusters) >= d.config.MaxClusters {
			d.dropLeastRecent()
		}

		heap.Push(&d.stamped, stampedCluster{
			cluster: c,
			stamp:   c.lastUsed.Load(),
		})
	}

	d.clusters[c.id] = c
}

// dropLeastRecent drops the least recently used cluster. Touches don't
// reorder the heap, so a top cluster used since it was stamped is
// restamped and sifted down first; the first top that wasn't used since is
// older than every other cluster.
func (d *drain) dropLeastRecent() {
	for {
		top := &d.stamped[0]
		if used := top.cluster.lastUsed.Load(); used != top.stamp {
			top.stamp = used
			heap.Fix(&d.stamped, 0)

			continue
		}

		dropped, _ := heap.Pop(&d.stamped).(stampedCluster)
		delete(d.clusters, dropped.cluster.id)

		return
	}
}

// touch marks c as the most recently used cluster. It needs only the read
// lock.
func (d *drain) touch(c *cluster) {
	if d.config.MaxClusters == 0 {
		return
	}

	// A concurrent touch may have stored a newer stamp since this one was
	// taken; keep the newest.
	stamp := d.used.Add(1)
	for {
		last := c.lastUsed.Load()
		if last >= stamp || c.lastUsed.CompareAndSwap(last, stamp) {
			return
		}
	}
}

// fits reports whether tokens fit c's template as it is: each token equals
// the template's or the template has a parameter there.
func (d *drain) fits(c *cluster, tokens []string) bool {
	for i, token := range tokens {
		if token != c.tokens[i] && c.tokens[i] != d.config.ParamString {
			return false
		}
	}

	return true
}

// leastRecentFirst returns the clusters from least to most recently used,
// or by ID without MaxClusters, where recency isn't tracked.
func (d *drain) leastRecentFirst() []*cluster {
	clusters := slices.Collect(maps.Values(d.clusters))

	slices.SortFunc(clusters, func(a, b *cluster) int {
		return cmp.Or(cmp.Compare(a.lastUsed.Load(), b.lastUsed.Load()), cmp.Compare(a.id, b.id))
	})

	return clusters
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
