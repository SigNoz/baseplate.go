package expr

import (
	"testing"

	upstream "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The cases are the SigNoz/expr fork's own test suite (v1.17.7-beta), each
// compiled with expr.Env(vars) and run, plus cases for what the suite
// doesn't cover. The expected results and errors were generated once by
// running every case on the fork. Through the shim, upstream expr gives the
// same results and the same errors, at the same positions in the source.
func TestShim(t *testing.T) {
	testCases := []struct {
		name          string
		expression    string
		vars          map[string]any
		expected      any
		expectedError *tError
	}{
		{
			name:       "CombinedExamples_Case1CheckoutServiceCriticalErrorInProduction_Expression1ProductionCritical",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "error_type": "timeout", "service": map[string]any{"name": "checkout"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "CombinedExamples_Case1CheckoutServiceCriticalErrorInProduction_Expression2CheckoutTeamCases",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "error_type": "timeout", "service": map[string]any{"name": "checkout"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "CombinedExamples_Case1CheckoutServiceCriticalErrorInProduction_Expression3PaymentServiceAllEnvs",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "error_type": "timeout", "service": map[string]any{"name": "checkout"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case1CheckoutServiceCriticalErrorInProduction_Expression4DevEnvironment",
			expression: `deployment.environment = "development"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "error_type": "timeout", "service": map[string]any{"name": "checkout"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case1CheckoutServiceCriticalErrorInProduction_Expression5InfrastructureCritical",
			expression: `type = "infrastructure" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "error_type": "timeout", "service": map[string]any{"name": "checkout"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case2PaymentServiceWarningInStaging_Expression1ProductionCritical",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "team": "payment", "threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case2PaymentServiceWarningInStaging_Expression2CheckoutTeamCases",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "team": "payment", "threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case2PaymentServiceWarningInStaging_Expression3PaymentServiceAllEnvs",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "team": "payment", "threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case2PaymentServiceWarningInStaging_Expression4DevEnvironment",
			expression: `deployment.environment = "development"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "team": "payment", "threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case2PaymentServiceWarningInStaging_Expression5InfrastructureCritical",
			expression: `type = "infrastructure" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "team": "payment", "threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case3DatabaseInfrastructureCriticalCase_Expression1ProductionCritical",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "postgres-primary"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   true,
		},
		{
			name:       "CombinedExamples_Case3DatabaseInfrastructureCriticalCase_Expression2CheckoutTeamCases",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "postgres-primary"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case3DatabaseInfrastructureCriticalCase_Expression3PaymentServiceAllEnvs",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "postgres-primary"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case3DatabaseInfrastructureCriticalCase_Expression4DevEnvironment",
			expression: `deployment.environment = "development"`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "postgres-primary"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case3DatabaseInfrastructureCriticalCase_Expression5InfrastructureCritical",
			expression: `type = "infrastructure" AND threshold.name = "critical"`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "postgres-primary"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   true,
		},
		{
			name:       "CombinedExamples_Case4InventoryServiceInfoInDevelopment_Expression1ProductionCritical",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "inventory"}, "team": "inventory", "threshold": map[string]any{"name": "info"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case4InventoryServiceInfoInDevelopment_Expression2CheckoutTeamCases",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "inventory"}, "team": "inventory", "threshold": map[string]any{"name": "info"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case4InventoryServiceInfoInDevelopment_Expression3PaymentServiceAllEnvs",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "inventory"}, "team": "inventory", "threshold": map[string]any{"name": "info"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "CombinedExamples_Case4InventoryServiceInfoInDevelopment_Expression4DevEnvironment",
			expression: `deployment.environment = "development"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "inventory"}, "team": "inventory", "threshold": map[string]any{"name": "info"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "CombinedExamples_Case4InventoryServiceInfoInDevelopment_Expression5InfrastructureCritical",
			expression: `type = "infrastructure" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "inventory"}, "team": "inventory", "threshold": map[string]any{"name": "info"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "NestedLabels_CloudInfrastructureLabels",
			expression: `cloud.provider = "aws" AND cloud.region = "us-east-1" AND cloud.availability_zone contains "us-east-1"`,
			vars:       map[string]any{"cloud": map[string]any{"availability_zone": "us-east-1a", "provider": "aws", "region": "us-east-1"}},
			expected:   true,
		},
		{
			name:       "NestedLabels_DatabaseQueryAttributes",
			expression: `db.system = "postgresql" AND db.name = "orders_db" AND db.operation IN ["INSERT", "UPDATE", "DELETE"]`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db", "operation": "UPDATE", "system": "postgresql"}},
			expected:   true,
		},
		{
			name:       "NestedLabels_HTTPRequestAttributes",
			expression: `http.method = "POST" AND http.route = "/api/checkout" AND http.status_code IN ["500", "502", "503"]`,
			vars:       map[string]any{"http": map[string]any{"method": "POST", "route": "/api/checkout", "status_code": "502"}},
			expected:   true,
		},
		{
			name:       "NestedLabels_NestedKubernetesLabels",
			expression: `k8s.pod.name = "api-pod-123" AND k8s.namespace.name = "production"`,
			vars:       map[string]any{"k8s": map[string]any{"deployment": map[string]any{"name": "api-deployment"}, "namespace": map[string]any{"name": "production"}, "pod": map[string]any{"name": "api-pod-123"}}},
			expected:   true,
		},
		{
			name:       "NestedLabels_ServiceAttributesWithVersion",
			expression: `service.name = "checkout" AND service.version = "v2.1.0" AND service.environment = "production"`,
			vars:       map[string]any{"service": map[string]any{"environment": "production", "name": "checkout", "version": "v2.1.0"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression1CriticalProductionCaseMatches",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "threshold": map[string]any{"name": "critical"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression1CriticalStagingCaseDoesnTMatch",
			expression: `deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "threshold": map[string]any{"name": "critical"}},
			expected:   false,
		},
		{
			name:       "DetailedExamples_Expression2CheckoutCriticalCaseMatches",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"service": map[string]any{"name": "checkout"}, "threshold": map[string]any{"name": "critical"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression2CheckoutInfoCaseDoesnTMatch",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"service": map[string]any{"name": "checkout"}, "threshold": map[string]any{"name": "info"}},
			expected:   false,
		},
		{
			name:       "DetailedExamples_Expression2CheckoutWarningCaseMatches",
			expression: `service.name = "checkout" AND threshold.name IN ["critical", "warning"]`,
			vars:       map[string]any{"service": map[string]any{"name": "checkout"}, "threshold": map[string]any{"name": "warning"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression3PaymentCriticalCaseMatches",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "threshold": map[string]any{"name": "critical"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression3PaymentWarningInProductionMatches",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "payment"}, "threshold": map[string]any{"name": "warning"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression3PaymentWarningInStagingDoesnTMatch",
			expression: `service.name = "payment" AND (threshold.name = "critical" OR (threshold.name = "warning" AND deployment.environment = "production"))`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "staging"}, "service": map[string]any{"name": "payment"}, "threshold": map[string]any{"name": "warning"}},
			expected:   false,
		},
		{
			name:       "DetailedExamples_Expression4DevelopmentEnvironmentMatches",
			expression: `deployment.environment = "development"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "development"}, "service": map[string]any{"name": "any-service"}},
			expected:   true,
		},
		{
			name:       "DetailedExamples_Expression5InfrastructureCriticalCaseMatches",
			expression: `type = "infrastructure" AND threshold.name = "critical"`,
			vars:       map[string]any{"threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   true,
		},
		{
			name:       "DynamicLabels_DatabaseOperation",
			expression: `db.name = "orders_db" AND db.operation = "SELECT"`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db", "operation": "SELECT", "system": "postgresql"}},
			expected:   true,
		},
		{
			name:       "DynamicLabels_HTTPMethodAndStatusCode",
			expression: `http.method = "POST" AND http.status_code = "500"`,
			vars:       map[string]any{"http": map[string]any{"method": "POST", "status_code": "500"}},
			expected:   true,
		},
		{
			name:       "DynamicLabels_HostNameINListWithRegion",
			expression: `host.name IN ["prod-server-1", "prod-server-2"] AND cloud.region = "us-east-1"`,
			vars:       map[string]any{"cloud": map[string]any{"region": "us-east-1"}, "host": map[string]any{"name": "prod-server-1"}},
			expected:   true,
		},
		{
			name:       "DynamicLabels_HostNameNotInList",
			expression: `host.name IN ["prod-server-1", "prod-server-2"] AND cloud.region = "us-east-1"`,
			vars:       map[string]any{"cloud": map[string]any{"region": "us-east-1"}, "host": map[string]any{"name": "prod-server-3"}},
			expected:   false,
		},
		{
			name:       "DynamicLabels_KubernetesProductionNamespaceWithAPIService",
			expression: `k8s.namespace.name = "production" AND service.name CONTAINS "api"`,
			vars:       map[string]any{"k8s": map[string]any{"namespace": map[string]any{"name": "production"}}, "service": map[string]any{"name": "checkout-api"}},
			expected:   true,
		},
		{
			name:       "DynamicLabels_KubernetesStagingNamespaceShouldNotMatchProduction",
			expression: `k8s.namespace.name = "production" AND service.name CONTAINS "api"`,
			vars:       map[string]any{"k8s": map[string]any{"namespace": map[string]any{"name": "staging"}}, "service": map[string]any{"name": "checkout-api"}},
			expected:   false,
		},
		{
			name:       "EdgeCases_BooleanLabelValues",
			expression: "is_critical = true",
			vars:       map[string]any{"is_critical": true},
			expected:   true,
		},
		{
			name:       "EdgeCases_CaseSensitivityInLabelNames",
			expression: `service.name = "api"`,
			vars:       map[string]any{"service": map[string]any{"Name": "api"}},
			expected:   false,
		},
		{
			name:       "EdgeCases_ComplexINExpressionWithMixedTypes",
			expression: `threshold.name IN ["critical", "warning", "error"]`,
			vars:       map[string]any{"threshold": map[string]any{"name": "warning"}},
			expected:   true,
		},
		{
			name:       "EdgeCases_EmptyLabelsMap",
			expression: `service.name = "api"`,
			vars:       map[string]any{},
			expectedError: &tError{
				message: "unknown name service",
				from:    0,
				to:      7,
				line:    1,
				column:  1,
			},
		},
		{
			name:       "EdgeCases_LabelExistsButIsNil",
			expression: "service.name != nil",
			vars:       map[string]any{"service": map[string]any{"name": nil}},
			expected:   false,
		},
		{
			name:       "EdgeCases_MissingNestedPath",
			expression: `k8s.pod.name = "test"`,
			vars:       map[string]any{"k8s": map[string]any{}},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "cannot fetch name from <nil>",
				from:    8,
				to:      12,
				line:    1,
				column:  9,
			},
		},
		{
			name:       "EdgeCases_NumericStringComparison",
			expression: `http.status_code = "200"`,
			vars:       map[string]any{"http": map[string]any{"status_code": "200"}},
			expected:   true,
		},
		{
			name:       "EdgeCases_PartialNestedPathExists",
			expression: "service.version.major = 2",
			vars:       map[string]any{"service": map[string]any{"name": "api"}},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "cannot fetch major from <nil>",
				from:    16,
				to:      21,
				line:    1,
				column:  17,
			},
		},
		{
			name:       "EdgeCases_SpecialCharactersInLabelValues",
			expression: `service.name = "api-service@v2.0"`,
			vars:       map[string]any{"service": map[string]any{"name": "api-service@v2.0"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ANDOperatorBothTrue",
			expression: `service.name = "api" AND environment = "production"`,
			vars:       map[string]any{"environment": "production", "service": map[string]any{"name": "api"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ANDOperatorOneFalse",
			expression: `service.name = "api" AND environment = "production"`,
			vars:       map[string]any{"environment": "staging", "service": map[string]any{"name": "api"}},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_ComplexExpressionWithParenthesesMatch",
			expression: `(service.name = "api" OR service.name = "frontend") AND environment = "production"`,
			vars:       map[string]any{"environment": "production", "service": map[string]any{"name": "api"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ComplexExpressionWithParenthesesNoMatch",
			expression: `(service.name = "api" OR service.name = "frontend") AND environment = "production"`,
			vars:       map[string]any{"environment": "production", "service": map[string]any{"name": "backend"}},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_EnvironmentInequalityMatch",
			expression: `environment != "development"`,
			vars:       map[string]any{"environment": "production"},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_EnvironmentInequalityNoMatch",
			expression: `environment != "development"`,
			vars:       map[string]any{"environment": "development"},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_OROperatorBothFalse",
			expression: `service.name = "api" OR environment = "production"`,
			vars:       map[string]any{"environment": "staging", "service": map[string]any{"name": "frontend"}},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_OROperatorBothTrue",
			expression: `service.name = "api" OR environment = "production"`,
			vars:       map[string]any{"environment": "production", "service": map[string]any{"name": "api"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_OROperatorOneTrue",
			expression: `service.name = "api" OR environment = "production"`,
			vars:       map[string]any{"environment": "production", "service": map[string]any{"name": "frontend"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ServiceNameCONTAINSMatch",
			expression: `service.name CONTAINS "auth"`,
			vars:       map[string]any{"service": map[string]any{"name": "user-auth-service"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ServiceNameContainsNoMatch",
			expression: `service.name contains "auth"`,
			vars:       map[string]any{"service": map[string]any{"name": "inventory-service"}},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_ServiceNameEqualityMatch",
			expression: `service.name = "api"`,
			vars:       map[string]any{"service": map[string]any{"name": "api"}},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_ServiceNameEqualityNoMatch",
			expression: `service.name = "api"`,
			vars:       map[string]any{"service": map[string]any{"name": "frontend"}},
			expected:   false,
		},
		{
			name:       "SyntaxOperators_SeverityInListMatch",
			expression: `severity in ["critical", "warning"]`,
			vars:       map[string]any{"severity": "critical"},
			expected:   true,
		},
		{
			name:       "SyntaxOperators_SeverityInListNoMatch",
			expression: `severity in ["critical", "warning"]`,
			vars:       map[string]any{"severity": "info"},
			expected:   false,
		},
		{
			name:       "ProvidedLabels_CaseNameMatching",
			expression: `alertname = "High CPU Usage"`,
			vars:       map[string]any{"alertname": "High CPU Usage"},
			expected:   true,
		},
		{
			name:       "ProvidedLabels_CriticalThresholdWithApplicationType",
			expression: `threshold.name = "critical" AND type = "application"`,
			vars:       map[string]any{"threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "ProvidedLabels_WarningThresholdShouldNotMatchCriticalExpression",
			expression: `threshold.name = "critical" AND type = "application"`,
			vars:       map[string]any{"threshold": map[string]any{"name": "warning"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "Scenarios_BusinessHoursCriticalCases",
			expression: `business_hours = true AND threshold.name = "critical" AND type = "application"`,
			vars:       map[string]any{"business_hours": true, "service": map[string]any{"name": "checkout"}, "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "Scenarios_DatabaseConnectionPoolExhausted",
			expression: `type = "infrastructure" AND db.name != "" AND deployment.environment = "production" AND threshold.name = "critical"`,
			vars:       map[string]any{"db": map[string]any{"name": "orders_db"}, "deployment": map[string]any{"environment": "production"}, "metric": map[string]any{"name": "connection_pool_exhausted"}, "threshold": map[string]any{"name": "critical"}, "type": "infrastructure"},
			expected:   true,
		},
		{
			name:       "Scenarios_HighLatencyInPaymentService",
			expression: `service.name = "payment" AND metric.name = "latency" AND threshold.name IN ["warning", "critical"]`,
			vars:       map[string]any{"metric": map[string]any{"name": "latency"}, "service": map[string]any{"name": "payment"}, "threshold": map[string]any{"name": "critical"}, "value": "2500ms"},
			expected:   true,
		},
		{
			name:       "Scenarios_KubernetesPodRestartLoop",
			expression: `k8s.pod.restart_count > 5 OR (metric.name = "pod_restarts" AND threshold.name = "critical")`,
			vars:       map[string]any{"k8s": map[string]any{"pod": map[string]any{"name": "checkout-api-xyz", "restart_count": 10}}, "metric": map[string]any{"name": "pod_restarts"}, "threshold": map[string]any{"name": "critical"}},
			expected:   true,
		},
		{
			name:       "Scenarios_MultiRegionFailureDetection",
			expression: `(cloud.region = "us-east-1" OR cloud.region = "eu-west-1") AND threshold.name = "critical" AND service.name = "api"`,
			vars:       map[string]any{"cloud": map[string]any{"region": "us-east-1"}, "service": map[string]any{"name": "api"}, "threshold": map[string]any{"name": "critical"}},
			expected:   true,
		},
		{
			name:       "MatchExample_CheckoutTeamWithApplicationType",
			expression: `team = "checkout" AND type = "application"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "MatchExample_CriticalThresholdInProduction",
			expression: `threshold.name = "critical" AND deployment.environment = "production"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "MatchExample_TeamEqualsCheckout",
			expression: `team = "checkout"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   true,
		},
		{
			name:       "MatchExample_WarningThresholdShouldNotMatch",
			expression: `threshold.name = "warning" AND deployment.environment = "production"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "MatchExample_WrongTeamShouldNotMatch",
			expression: `team = "payment"`,
			vars:       map[string]any{"deployment": map[string]any{"environment": "production"}, "team": "checkout", "threshold": map[string]any{"name": "critical"}, "type": "application"},
			expected:   false,
		},
		{
			name:       "SQLStyleComplexScenarios_CaseWithUppercaseOperators",
			expression: "severity IN [\"critical\", \"high\"] AND\n\t\t\t            deployment.environment = \"production\" AND\n\t\t\t            (service.name REGEXP \"^api-\" OR service.name REGEXP \"^auth-\") AND\n\t\t\t            NOT (alert.silenced = true OR alert.acknowledged = true)",
			vars:       map[string]any{"alert": map[string]any{"acknowledged": false, "silenced": false}, "deployment": map[string]any{"environment": "production"}, "service": map[string]any{"name": "api-gateway"}, "severity": "critical"},
			expected:   true,
		},
		{
			name:       "SQLStyleComplexScenarios_ComplexUserPermissionCheck",
			expression: "(user.role IN [\"admin\", \"superuser\"]) OR \n\t\t\t            (user.role = \"moderator\" AND user.permissions CONTAINS \"edit\") OR\n\t\t\t            (user.role = \"user\" AND resource.owner = user.id AND resource.public = false)",
			vars:       map[string]any{"resource": map[string]any{"owner": 123, "public": false}, "user": map[string]any{"id": 123, "permissions": []any{"read", "comment"}, "role": "user"}},
			expected:   true,
		},
		{
			name:       "SQLStyleComplexScenarios_EmailFilterWithMultipleConditions",
			expression: "(sender REGEXP \"@(spam|junk|trash)\\\\.com$\" OR \n\t\t\t             subject CONTAINS \"WIN NOW\" OR \n\t\t\t             subject CONTAINS \"FREE MONEY\") AND\n\t\t\t            NOT (sender IN [\"whitelist@example.com\", \"trusted@company.com\"]) AND\n\t\t\t            attachments > 0",
			vars:       map[string]any{"attachments": 2, "sender": "offer@spam.com", "subject": "Regular email"},
			expected:   true,
		},
		{
			name:       "SQLStyleComplexScenarios_FeatureFlagWithComplexTargeting",
			expression: "(user.subscription = \"premium\" OR user.subscription = \"enterprise\") AND\n\t\t\t            user.region IN [\"US\", \"CA\", \"UK\"] AND\n\t\t\t            (user.beta_tester = true OR user.employee = true OR percentage < 10)",
			vars:       map[string]any{"percentage": 5, "user": map[string]any{"beta_tester": false, "employee": false, "region": "US", "subscription": "premium"}},
			expected:   true,
		},
		{
			name:       "SQLStyleComplexScenarios_FilterWithNestedConditions",
			expression: "level IN [\"ERROR\", \"FATAL\"] AND\n\t\t\t            (message CONTAINS \"timeout\" OR message CONTAINS \"connection refused\") AND\n\t\t\t            service.name != \"health-check\" AND\n\t\t\t            NOT (tags CONTAINS \"expected-error\" OR tags CONTAINS \"ignore\")",
			vars:       map[string]any{"level": "ERROR", "message": "Database connection refused", "service": map[string]any{"name": "api-service"}, "tags": []any{"database", "critical"}},
			expectedError: &tError{
				message: "invalid operation: CONTAINS (mismatched types []interface {} and string)",
				from:    196,
				to:      204,
				line:    4,
				column:  26,
			},
		},
		{
			name:       "SQLStyleComplexScenarios_SQLLikeWHEREClauseSimulation",
			expression: "status IN [\"active\", \"pending\"] AND \n\t\t\t            created_date >= \"2024-01-01\" AND \n\t\t\t            (category = \"electronics\" OR category = \"computers\") AND\n\t\t\t            price BETWEEN 100 AND 1000",
			vars:       map[string]any{"category": "electronics", "created_date": "2024-03-15", "price": 599.99, "status": "active"},
			expectedError: &tError{
				message: `unexpected token Identifier("BETWEEN")`,
				from:    179,
				to:      186,
				line:    4,
				column:  22,
			},
		},
		{
			name:       "SQLStyleEqualityOperators_MixSQLAndStandardEquals",
			expression: `firstName = "John" AND lastName == "Doe"`,
			vars:       map[string]any{"firstName": "John", "lastName": "Doe"},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_MixSQLAndStandardNotEquals",
			expression: `status != "deleted" AND type != "temp"`,
			vars:       map[string]any{"status": "active", "type": "permanent"},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLEqualsOperatorNestedField",
			expression: `user.status = "active"`,
			vars:       map[string]any{"user": map[string]any{"status": "active"}},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLEqualsOperatorNumberMatch",
			expression: "age = 25",
			vars:       map[string]any{"age": 25},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLEqualsOperatorStringMatch",
			expression: `name = "John"`,
			vars:       map[string]any{"name": "John"},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLEqualsOperatorStringNoMatch",
			expression: `name = "Jane"`,
			vars:       map[string]any{"name": "John"},
			expected:   false,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLNotEqualsOperatorNestedField",
			expression: `product.type != "premium"`,
			vars:       map[string]any{"product": map[string]any{"type": "basic"}},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLNotEqualsOperatorNumber",
			expression: "count != 0",
			vars:       map[string]any{"count": 5},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLNotEqualsOperatorStringDifferent",
			expression: `name != "Jane"`,
			vars:       map[string]any{"name": "John"},
			expected:   true,
		},
		{
			name:       "SQLStyleEqualityOperators_SQLNotEqualsOperatorStringSame",
			expression: `name != "John"`,
			vars:       map[string]any{"name": "John"},
			expected:   false,
		},
		{
			name:       "SQLStyleErrorCases_INWithNonArray",
			expression: `status IN "active"`,
			vars:       map[string]any{"status": "active"},
			expectedError: &tError{
				message: "invalid operation: IN (mismatched types string and string)",
				from:    7,
				to:      9,
				line:    1,
				column:  8,
			},
		},
		{
			name: "SQLStyleErrorCases_InvalidOperatorCombination",
			// The repeated NOT is the point; split so it isn't read as a typo.
			expression: "status NOT" + ` NOT IN ["active"]`,
			vars:       map[string]any{"status": "active"},
			expectedError: &tError{
				message: `unexpected token Operator("NOT")`,
				from:    11,
				to:      14,
				line:    1,
				column:  12,
			},
		},
		{
			name:       "SQLStyleErrorCases_REGEXPWithInvalidPattern",
			expression: `text REGEXP "[invalid"`,
			vars:       map[string]any{"text": "test"},
			expectedError: &tError{
				message: "error parsing regexp: missing closing ]: `[invalid`",
				from:    5,
				to:      11,
				line:    1,
				column:  6,
			},
		},
		{
			name:       "SQLStyleErrorCases_TypeMismatchWithCONTAINS",
			expression: `count CONTAINS "5"`,
			vars:       map[string]any{"count": 12345},
			expectedError: &tError{
				message: "invalid operation: CONTAINS (mismatched types int and string)",
				from:    6,
				to:      14,
				line:    1,
				column:  7,
			},
		},
		{
			name:       "SQLStyleErrorCases_UndefinedFieldAccess",
			expression: `user.name = "John" AND user.age > 18`,
			vars:       map[string]any{"user": map[string]any{"name": "John"}},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: <nil> > int",
				from:    32,
				to:      33,
				line:    1,
				column:  33,
			},
		},
		{
			name:       "SQLStyleInOperator_INWithNestedField",
			expression: `user.role IN ["admin", "moderator", "editor"]`,
			vars:       map[string]any{"user": map[string]any{"role": "editor"}},
			expected:   true,
		},
		{
			name:       "SQLStyleInOperator_MixUppercaseAndLowercaseIN",
			expression: `(status in ["active", "pending"]) AND (priority IN [1, 2, 3])`,
			vars:       map[string]any{"priority": 2, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleInOperator_MultipleINConditions",
			expression: `department IN ["sales", "marketing"] AND level IN [3, 4, 5]`,
			vars:       map[string]any{"department": "sales", "level": 4},
			expected:   true,
		},
		{
			name:       "SQLStyleInOperator_UppercaseINEmptyArray",
			expression: "status IN []",
			vars:       map[string]any{"status": "active"},
			expected:   false,
		},
		{
			name:       "SQLStyleInOperator_UppercaseINNumberFound",
			expression: "priority IN [1, 2, 3]",
			vars:       map[string]any{"priority": 2},
			expected:   true,
		},
		{
			name:       "SQLStyleInOperator_UppercaseINStringFound",
			expression: `status IN ["active", "pending", "processing"]`,
			vars:       map[string]any{"status": "pending"},
			expected:   true,
		},
		{
			name:       "SQLStyleInOperator_UppercaseINStringNotFound",
			expression: `status IN ["active", "pending", "processing"]`,
			vars:       map[string]any{"status": "canceled"},
			expected:   false,
		},
		{
			name:       "SQLStyleInOperator_UppercaseNOTINValueInList",
			expression: `status NOT IN ["deleted", "archived", "hidden"]`,
			vars:       map[string]any{"status": "deleted"},
			expected:   false,
		},
		{
			name:       "SQLStyleInOperator_UppercaseNOTINValueNotInList",
			expression: `status NOT IN ["deleted", "archived", "hidden"]`,
			vars:       map[string]any{"status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_ComplexAND_ORExpression",
			expression: `(status = "active" AND role = "admin") OR superuser = true`,
			vars:       map[string]any{"role": "user", "status": "active", "superuser": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_MixAllLogicalOperators",
			expression: "(active = true && verified = true) AND (premium = true || trial = true) OR override = true",
			vars:       map[string]any{"active": true, "override": false, "premium": false, "trial": true, "verified": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_MixUppercaseAndLowercaseAND",
			expression: "active = true AND verified = true and enabled = true",
			vars:       map[string]any{"active": true, "enabled": true, "verified": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_MixUppercaseAndLowercaseOR",
			expression: `status = "pending" OR status = "processing" or status = "queued"`,
			vars:       map[string]any{"status": "queued"},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_NestedAND_ORWithParentheses",
			expression: `enabled = true AND (role = "admin" OR role = "moderator")`,
			vars:       map[string]any{"enabled": true, "role": "moderator"},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseANDBothFalse",
			expression: "active = false AND verified = false",
			vars:       map[string]any{"active": false, "verified": false},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseANDBothTrue",
			expression: "active = true AND verified = true",
			vars:       map[string]any{"active": true, "verified": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseANDFirstFalse",
			expression: "active = false AND verified = true",
			vars:       map[string]any{"active": false, "verified": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseANDSecondFalse",
			expression: "active = true AND verified = false",
			vars:       map[string]any{"active": true, "verified": false},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseORBothFalse",
			expression: "premium = false OR trial = false",
			vars:       map[string]any{"premium": false, "trial": false},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseORBothTrue",
			expression: "premium = true OR trial = true",
			vars:       map[string]any{"premium": true, "trial": true},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseORFirstTrue",
			expression: "premium = true OR trial = false",
			vars:       map[string]any{"premium": true, "trial": false},
			expected:   true,
		},
		{
			name:       "SQLStyleLogicalOperators_UppercaseORSecondTrue",
			expression: "premium = false OR trial = true",
			vars:       map[string]any{"premium": false, "trial": true},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorCaseSensitivity_AllLowercaseOperators",
			expression: `status = "active" and priority in [1, 2, 3] and message not contains "error"`,
			vars:       map[string]any{"message": "All good", "priority": 2, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorCaseSensitivity_AllUppercaseSQLOperators",
			expression: `status = "active" AND priority IN [1, 2, 3] AND message NOT CONTAINS "error"`,
			vars:       map[string]any{"message": "All good", "priority": 2, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorCaseSensitivity_MixedCaseOperators",
			expression: `status = "active" AND priority in [1, 2, 3] and message NOT CONTAINS "error"`,
			vars:       map[string]any{"message": "All good", "priority": 2, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorCaseSensitivity_REGEXPInUppercase",
			expression: `email REGEXP "^[^@]+@[^@]+\\.[^@]+$"`,
			vars:       map[string]any{"email": "test@example.com"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorCaseSensitivity_RegexpInLowercase",
			expression: `email regexp "^[^@]+@[^@]+\\.[^@]+$"`,
			vars:       map[string]any{"email": "test@example.com"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorOptimizations_ConstantFoldingWithSQLEquals",
			expression: `5 = 5 AND status = "active"`,
			vars:       map[string]any{"status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorOptimizations_ConstantFoldingWithSQLNotEquals",
			expression: `"a" != "b" AND enabled = true`,
			vars:       map[string]any{"enabled": true},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorOptimizations_INWithSingleElement",
			expression: `status IN ["active"]`,
			vars:       map[string]any{"status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorOptimizations_NOTINWithEmptyArray",
			expression: "status NOT IN []",
			vars:       map[string]any{"status": "anything"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorPrecedence_ANDHasHigherPrecedenceThanOR",
			expression: `status = "active" OR status = "pending" AND priority = 1`,
			vars:       map[string]any{"priority": 5, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorPrecedence_ComparisonOperatorsHaveSamePrecedence",
			expression: `age >= 18 AND age <= 65 AND status = "active"`,
			vars:       map[string]any{"age": 25, "status": "active"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorPrecedence_ComplexPrecedenceWithMixedOperators",
			expression: `role IN ["admin", "moderator"] AND active = true OR override = true`,
			vars:       map[string]any{"active": true, "override": true, "role": "user"},
			expected:   true,
		},
		{
			name:       "SQLStyleOperatorPrecedence_ParenthesesOverridePrecedence",
			expression: `(status = "active" OR status = "pending") AND priority = 1`,
			vars:       map[string]any{"priority": 5, "status": "active"},
			expected:   false,
		},
		{
			name:       "SQLStyleOperatorPrecedence_UppercaseANDHasHigherPrecedenceThanOR",
			expression: "verified = false OR active = true AND enabled = true",
			vars:       map[string]any{"active": true, "enabled": true, "verified": false},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_ComplexStringOperatorExpression",
			expression: `(email REGEXP "@company\\.com$") AND (name CONTAINS "Admin" OR role = "admin")`,
			vars:       map[string]any{"email": "john@company.com", "name": "John Administrator", "role": "user"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_LowercaseRegexpOperator",
			expression: `phone regexp "^\\+?[1-9]\\d{1,14}$"`,
			vars:       map[string]any{"phone": "+1234567890"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_MixUppercaseAndLowercaseStringOperators",
			expression: `(title contains "important") AND (body CONTAINS "urgent")`,
			vars:       map[string]any{"body": "This is urgent!", "title": "important notice"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_NOTCONTAINS",
			expression: `message NOT CONTAINS "spam"`,
			vars:       map[string]any{"message": "Important notification"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_NOTREGEXP",
			expression: `username NOT REGEXP "^[0-9]"`,
			vars:       map[string]any{"username": "john123"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_UppercaseCONTAINSCaseSensitive",
			expression: `message CONTAINS "Error"`,
			vars:       map[string]any{"message": "error occurred"},
			expected:   false,
		},
		{
			name:       "SQLStyleStringOperators_UppercaseCONTAINSSubstringFound",
			expression: `message CONTAINS "error"`,
			vars:       map[string]any{"message": "Critical error occurred"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_UppercaseCONTAINSSubstringNotFound",
			expression: `message CONTAINS "warning"`,
			vars:       map[string]any{"message": "Everything is fine"},
			expected:   false,
		},
		{
			name:       "SQLStyleStringOperators_UppercaseREGEXPPatternMatch",
			expression: `email REGEXP "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"`,
			vars:       map[string]any{"email": "user@example.com"},
			expected:   true,
		},
		{
			name:       "SQLStyleStringOperators_UppercaseREGEXPPatternNoMatch",
			expression: `email REGEXP "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"`,
			vars:       map[string]any{"email": "invalid-email"},
			expected:   false,
		},
		{
			name:       "StaticLabels_ApplicationType",
			expression: `type = "application"`,
			vars:       map[string]any{"team": "checkout", "type": "application"},
			expected:   true,
		},
		{
			name:       "StaticLabels_CustomerFacingSLA",
			expression: `sla = "customer-facing"`,
			vars:       map[string]any{"sla": "customer-facing", "type": "business"},
			expected:   true,
		},
		{
			name:       "StaticLabels_InfrastructureAndPlatformTeam",
			expression: `type = "infrastructure" AND team = "platform"`,
			vars:       map[string]any{"team": "platform", "type": "infrastructure"},
			expected:   true,
		},
		{
			name:       "StaticLabels_InfrastructureButNotPlatformTeam",
			expression: `type = "infrastructure" AND team = "platform"`,
			vars:       map[string]any{"team": "payments", "type": "infrastructure"},
			expected:   false,
		},
		{
			name:       "MemberName_Regexp_NotRespelled",
			expression: `attributes.regexp = "x"`,
			vars:       map[string]any{"attributes": map[string]any{"regexp": "x"}},
			expected:   true,
		},
		{
			name:       "MemberName_In_NotRespelled",
			expression: `attributes.IN = "y"`,
			vars:       map[string]any{"attributes": map[string]any{"IN": "y"}},
			expected:   true,
		},
		{
			name:       "OptionalMemberName_And_NotRespelled",
			expression: `attributes?.AND = "z"`,
			vars:       map[string]any{"attributes": map[string]any{"AND": "z"}},
			expected:   true,
		},
		{
			name:       "MemberNames_ContainsAndNot_NotRespelled",
			expression: `attributes.contains = "w" AND attributes.not = "v"`,
			vars:       map[string]any{"attributes": map[string]any{"contains": "w", "not": "v"}},
			expected:   true,
		},
		{
			name:       "LetBinder_Equals_Kept",
			expression: "let x = 1; x = 1",
			vars:       map[string]any{},
			expected:   true,
		},
		{
			name:       "LetBinder_UsedInComparison_Kept",
			expression: "let limit = 3; count = limit",
			vars:       map[string]any{"count": 3},
			expected:   true,
		},
		{
			name:       "QuotedKeywords_NotRespelled",
			expression: `"a AND b = c" CONTAINS "AND"`,
			vars:       map[string]any{},
			expected:   true,
		},
		{
			name:       "SingleQuotedKeywords_NotRespelled",
			expression: "label = 'x = y OR z'",
			vars:       map[string]any{"label": "x = y OR z"},
			expected:   true,
		},
		{
			name:       "LowerCaseKeywords_Accepted",
			expression: `env = "prod" and region in ["eu"]`,
			vars:       map[string]any{"env": "prod", "region": "eu"},
			expected:   true,
		},
		{
			name:       "LowerCaseRegexp_Respelled",
			expression: `name regexp "^al"`,
			vars:       map[string]any{"name": "alpha"},
			expected:   true,
		},
		{
			name:       "RegexpOrContains_Respelled",
			expression: `name REGEXP "^al" OR name CONTAINS "ph"`,
			vars:       map[string]any{"name": "alpha"},
			expected:   true,
		},
		{
			name:       "MixedCaseAnd_SyntaxInvalid",
			expression: `env = "prod" And region = "eu"`,
			vars:       map[string]any{"env": "prod", "region": "eu"},
			expectedError: &tError{
				message: `unexpected token Identifier("And")`,
				from:    13,
				to:      16,
				line:    1,
				column:  14,
			},
		},
		{
			name:       "ComparisonOperators_NotRespelled",
			expression: "count <= 3 AND count >= 1 AND count != 2",
			vars:       map[string]any{"count": 3},
			expected:   true,
		},
		{
			name:       "NotWithParentheses_Negates",
			expression: `NOT (env = "prod")`,
			vars:       map[string]any{"env": "prod"},
			expected:   false,
		},
		{
			name:       "EqualsInPredicate_Respelled",
			expression: "all(items, # = 1)",
			vars:       map[string]any{"items": []any{1, 1}},
			expected:   true,
		},
		{
			name:       "TypeError_AfterAnd_PositionOnSource",
			expression: `env = "prod" AND count > "5"`,
			vars:       map[string]any{"count": 9, "env": "prod"},
			expectedError: &tError{
				message: "invalid operation: > (mismatched types int and string)",
				from:    23,
				to:      24,
				line:    1,
				column:  24,
			},
		},
		{
			name:       "TypeError_AfterNonASCII_PositionInCodePoints",
			expression: `city = "Zürich" AND count > "5"`,
			vars:       map[string]any{"city": "Zürich", "count": 9},
			expectedError: &tError{
				message: "invalid operation: > (mismatched types int and string)",
				from:    26,
				to:      27,
				line:    1,
				column:  27,
			},
		},
		{
			name:       "TypeError_AfterEmoji_PositionInCodePoints",
			expression: `tag = "🚀" AND count > "5"`,
			vars:       map[string]any{"count": 9, "tag": "🚀"},
			expectedError: &tError{
				message: "invalid operation: > (mismatched types int and string)",
				from:    20,
				to:      21,
				line:    1,
				column:  21,
			},
		},
		{
			name:       "TypeError_OnSecondLine_LineAndColumn",
			expression: "env = \"prod\"\nAND count > \"5\"",
			vars:       map[string]any{"count": 9, "env": "prod"},
			expectedError: &tError{
				message: "invalid operation: > (mismatched types int and string)",
				from:    23,
				to:      24,
				line:    2,
				column:  11,
			},
		},
		{
			name:       "RegexpInvalid_PositionOnSource",
			expression: `name REGEXP "[a" AND env = "prod"`,
			vars:       map[string]any{"env": "prod", "name": "alpha"},
			expectedError: &tError{
				message: "error parsing regexp: missing closing ]: `[a`",
				from:    5,
				to:      11,
				line:    1,
				column:  6,
			},
		},
		{
			name:       "DoubleEquals_SyntaxInvalid",
			expression: "count = = 1",
			vars:       map[string]any{"count": 1},
			expectedError: &tError{
				message: `unexpected token Operator("=")`,
				from:    8,
				to:      9,
				line:    1,
				column:  9,
			},
		},
		{
			name:       "TrailingAnd_UnexpectedEOF",
			expression: `env = "prod" AND`,
			vars:       map[string]any{"env": "prod"},
			expectedError: &tError{
				message: "unexpected token EOF",
				from:    15,
				to:      16,
				line:    1,
				column:  16,
			},
		},
		{
			name:       "MissingNestedField_RunFailed",
			expression: `user.name = "bravo" AND user.age > 18`,
			vars:       map[string]any{"user": map[string]any{"name": "bravo"}},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: <nil> > int",
				from:    33,
				to:      34,
				line:    1,
				column:  34,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newSource(testCase.expression)
			if err == nil {
				var program *vm.Program

				program, err = upstream.Compile(s.upstream, upstream.Env(testCase.vars))
				if err != nil {
					err = s.positionError(err, CodeTypeInvalid)
				}

				if err == nil {
					var got any

					got, err = upstream.Run(program, testCase.vars)
					if err == nil {
						require.Nil(t, testCase.expectedError)
						assert.Equal(t, testCase.expected, got)

						return
					}

					err = s.positionError(err, CodeRunFailed)
				}
			}

			require.NotNil(t, testCase.expectedError, "unexpected error: %v", err)
			tAssertError(t, testCase.expectedError, err)
		})
	}
}

func TestShim_Spelling(t *testing.T) {
	testCases := []struct {
		name             string
		source           string
		expectedUpstream string
	}{
		{
			name:             "Aliases_Respelled",
			source:           `a = 1 AND b IN [1] OR NOT c CONTAINS "x"`,
			expectedUpstream: `a == 1 and b in [1] or not c contains "x"`,
		},
		{
			name:             "Regexp_BothCases",
			source:           `a REGEXP "x" OR a regexp "y"`,
			expectedUpstream: `a matches "x" or a matches "y"`,
		},
		{
			name:             "QuotedText_Untouched",
			source:           `a = "b AND c = d"`,
			expectedUpstream: `a == "b AND c = d"`,
		},
		{
			name:             "MemberNames_Untouched",
			source:           `a.AND = 1 AND a?.regexp = 2`,
			expectedUpstream: `a.AND == 1 and a?.regexp == 2`,
		},
		{
			name:             "LetBinder_Untouched",
			source:           `let x = 1; x = 1`,
			expectedUpstream: `let x = 1; x == 1`,
		},
		{
			name:             "OtherOperators_Untouched",
			source:           `a <= 1 && b != 2 || c >= 3`,
			expectedUpstream: `a <= 1 && b != 2 || c >= 3`,
		},
		{
			name:             "MixedCase_Untouched",
			source:           `a And b`,
			expectedUpstream: `a And b`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newSource(testCase.source)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedUpstream, s.upstream)
		})
	}
}
