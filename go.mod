module github.com/BishopFox/cloudfox

go 1.24

toolchain go1.24.3

require (
	cloud.google.com/go/artifactregistry v1.17.1
	cloud.google.com/go/bigquery v1.68.0
	cloud.google.com/go/iam v1.5.2
	cloud.google.com/go/resourcemanager v1.10.6
	cloud.google.com/go/secretmanager v1.14.7
	cloud.google.com/go/storage v1.54.0
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1
	github.com/Azure/go-autorest/autorest v0.11.30
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13
	github.com/aquasecurity/table v1.10.0
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.30.1
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.27.1
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.34.0
	github.com/aws/aws-sdk-go-v2/service/athena v1.50.5
	github.com/aws/aws-sdk-go-v2/service/cloud9 v1.29.2
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.59.2
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.46.1
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.48.4
	github.com/aws/aws-sdk-go-v2/service/codeartifact v1.34.2
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.61.0
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.28.2
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.30.3
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.26.2
	github.com/aws/aws-sdk-go-v2/service/directoryservice v1.31.5
	github.com/aws/aws-sdk-go-v2/service/docdb v1.41.3
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.43.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.219.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.44.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.57.1
	github.com/aws/aws-sdk-go-v2/service/efs v1.35.3
	github.com/aws/aws-sdk-go-v2/service/eks v1.64.0
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.46.0
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.29.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.29.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.45.2
	github.com/aws/aws-sdk-go-v2/service/emr v1.49.0
	github.com/aws/aws-sdk-go-v2/service/fsx v1.53.4
	github.com/aws/aws-sdk-go-v2/service/glue v1.111.0
	github.com/aws/aws-sdk-go-v2/service/grafana v1.27.2
	github.com/aws/aws-sdk-go-v2/service/iam v1.42.0
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.35.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.71.2
	github.com/aws/aws-sdk-go-v2/service/lightsail v1.43.2
	github.com/aws/aws-sdk-go-v2/service/mq v1.29.0
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.46.3
	github.com/aws/aws-sdk-go-v2/service/organizations v1.38.3
	github.com/aws/aws-sdk-go-v2/service/ram v1.30.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.95.0
	github.com/aws/aws-sdk-go-v2/service/redshift v1.54.3
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.26.3
	github.com/aws/aws-sdk-go-v2/service/route53 v1.51.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.79.3
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.192.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.4
	github.com/aws/aws-sdk-go-v2/service/sfn v1.35.4
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.4
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.5
	github.com/aws/aws-sdk-go-v2/service/ssm v1.59.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19
	github.com/aws/smithy-go v1.22.3
	github.com/bishopfox/awsservicemap v1.0.4
	github.com/bishopfox/knownawsaccountslookup v0.0.0-20231228165844-c37ef8df33cb
	github.com/dominikbraun/graph v0.23.0
	github.com/fatih/color v1.18.0
	github.com/goccy/go-json v0.10.5
	github.com/googleapis/gax-go/v2 v2.14.2
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/kyokomi/emoji v2.2.4+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/afero v1.14.0
	github.com/spf13/cobra v1.9.1
	golang.org/x/crypto v0.38.0
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/auth v0.16.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.3.1 // indirect
	github.com/charmbracelet/x/ansi v0.9.2 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2/service/kms v1.38.3
	golang.org/x/oauth2 v0.30.0
	google.golang.org/api v0.233.0
	google.golang.org/genproto v0.0.0-20250519155744-55703ea1f237
	google.golang.org/protobuf v1.36.6
)

require (
	cloud.google.com/go v0.121.1 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.7 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/charmbracelet/bubbles v0.21.0
	github.com/charmbracelet/bubbletea v1.3.5
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/go-openapi/errors v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.21.10 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/neo4j/neo4j-go-driver/v5 v5.28.1
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.mongodb.org/mongo-driver v1.13.1 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250519155744-55703ea1f237 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250519155744-55703ea1f237 // indirect
	google.golang.org/grpc v1.72.1 // indirect
)
