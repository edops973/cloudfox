package aws

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/BishopFox/cloudfox/aws/sdk"
	"github.com/BishopFox/cloudfox/internal"
	"github.com/BishopFox/cloudfox/internal/aws/policy"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/bishopfox/awsservicemap"
	"github.com/sirupsen/logrus"
)

type ResourceTrustsModule struct {
	KMSClient        *sdk.KMSClientInterface
	APIGatewayClient *sdk.APIGatewayClientInterface
	EC2Client        *sdk.AWSEC2ClientInterface
	OpenSearchClient *sdk.OpenSearchClientInterface

	// General configuration data
	Caller             sts.GetCallerIdentityOutput
	AWSRegions         []string
	Goroutines         int
	WrapTable          bool
	AWSOutputType      string
	AWSTableCols       string
	AWSMFAToken        string
	AWSConfig          aws.Config
	AWSProfileProvided string
	AWSProfileStub     string
	CloudFoxVersion    string

	Resources2     []Resource2
	CommandCounter internal.CommandCounter

	output internal.OutputData2
	modLog *logrus.Entry
}

type Resource2 struct {
	AccountID             string
	Name                  string
	ARN                   string
	Region                string
	Policy                policy.Policy
	PolicyJSON            string
	ResourcePolicySummary string
	Public                string
	Interesting           string
	TrustedPrincipals     string
	TrustsCrossAccount    string
	TrustsAllAccounts     string
	HasConditions         string
}

func (m *ResourceTrustsModule) PrintResources(outputDirectory string, verbosity int, includeKms bool) {
	// These struct values are used by the output module
	m.output.Verbosity = verbosity
	m.output.Directory = outputDirectory
	m.output.CallingModule = "resource-trusts"
	m.modLog = internal.TxtLog.WithFields(logrus.Fields{
		"module": m.output.CallingModule,
	})

	if m.AWSProfileProvided == "" {
		m.AWSProfileStub = internal.BuildAWSPath(m.Caller)
	} else {
		m.AWSProfileStub = m.AWSProfileProvided
	}
	m.output.FilePath = filepath.Join(outputDirectory, "cloudfox-output", "aws", fmt.Sprintf("%s-%s", m.AWSProfileProvided, aws.ToString(m.Caller.Account)))

	fmt.Printf("[%s][%s] Enumerating Resources with resource policies for account %s.\n", cyan(m.output.CallingModule), cyan(m.AWSProfileStub), aws.ToString(m.Caller.Account))
	// if kms feature flag is enabled include kms in the supported services
	if includeKms {
		fmt.Printf("[%s][%s] Supported Services: APIGateway, CodeBuild, ECR, EFS, Glue, KMS, Lambda, Opensearch, SecretsManager, S3, SNS, SQS, VpcEndpoint\n",
			cyan(m.output.CallingModule), cyan(m.AWSProfileStub))
	} else {
		fmt.Printf("[%s][%s] Supported Services: APIGateway, CodeBuild, ECR, EFS, Glue, Lambda, Opensearch, SecretsManager, S3, SNS, "+
			"SQS, VpcEndpoint (KMS requires --include-kms feature flag)\n",
			cyan(m.output.CallingModule), cyan(m.AWSProfileStub))
	}
	wg := new(sync.WaitGroup)
	semaphore := make(chan struct{}, m.Goroutines)

	// Create a channel to signal the spinner aka task status goroutine to finish
	spinnerDone := make(chan bool)
	//fire up the task status spinner/updated
	go internal.SpinUntil(m.output.CallingModule, &m.CommandCounter, spinnerDone, "tasks")

	//create a channel to receive the objects
	dataReceiver := make(chan Resource2)

	// Create a channel to signal to stop
	receiverDone := make(chan bool)
	go m.Receiver(dataReceiver, receiverDone)

	for _, region := range m.AWSRegions {
		wg.Add(1)
		m.CommandCounter.Pending++
		go m.executeChecks(region, wg, semaphore, dataReceiver, includeKms)

	}
	wg.Add(1)
	go m.getS3Buckets(wg, semaphore, dataReceiver)

	wg.Wait()
	// Send a message to the spinner goroutine to close the channel and stop
	spinnerDone <- true
	<-spinnerDone
	// Send a message to the data receiver goroutine to close the channel and stop
	receiverDone <- true
	<-receiverDone

	// add - if struct is not empty do this. otherwise, dont write anything.
	m.output.Headers = []string{
		"Account",
		"ARN",
		"Public",
		"Interesting",
		"Resource Policy Summary",
	}

	// If the user specified table columns, use those.
	// If the user specified -o wide, use the wide default cols for this module.
	// Otherwise, use the hardcoded default cols for this module.
	var tableCols []string
	// If the user specified table columns, use those.
	if m.AWSTableCols != "" {
		// If the user specified wide as the output format, use these columns.
		// remove any spaces between any commas and the first letter after the commas
		m.AWSTableCols = strings.ReplaceAll(m.AWSTableCols, ", ", ",")
		m.AWSTableCols = strings.ReplaceAll(m.AWSTableCols, ",  ", ",")
		tableCols = strings.Split(m.AWSTableCols, ",")
	} else if m.AWSOutputType == "wide" {
		tableCols = []string{
			"Account",
			"ARN",
			"Public",
			"Interesting",
			"Resource Policy Summary",
		}
	} else {
		tableCols = []string{
			//"Account ID",
			"ARN",
			"Public",
			"Interesting",
			"Resource Policy Summary",
		}
	}

	// sort the table roles by Interesting
	sort.Slice(m.Resources2, func(i, j int) bool {
		return m.Resources2[j].Interesting > m.Resources2[i].Interesting
	})

	// Table rows
	for i := range m.Resources2 {
		m.output.Body = append(
			m.output.Body,
			[]string{
				aws.ToString(m.Caller.Account),
				m.Resources2[i].ARN,
				m.Resources2[i].Public,
				m.Resources2[i].Interesting,
				m.Resources2[i].ResourcePolicySummary,
			},
		)

	}
	if len(m.output.Body) > 0 {
		o := internal.OutputClient{
			Verbosity:     verbosity,
			CallingModule: m.output.CallingModule,
			Table: internal.TableClient{
				Wrap: m.WrapTable,
			},
		}
		o.Table.TableFiles = append(o.Table.TableFiles, internal.TableFile{
			Header:    m.output.Headers,
			Body:      m.output.Body,
			TableCols: tableCols,
			Name:      m.output.CallingModule,
		})
		o.PrefixIdentifier = m.AWSProfileStub
		o.Table.DirectoryName = filepath.Join(outputDirectory, "cloudfox-output", "aws", fmt.Sprintf("%s-%s", m.AWSProfileStub, aws.ToString(m.Caller.Account)))
		o.WriteFullOutput(o.Table.TableFiles, nil)
		//m.writeLoot(o.Table.DirectoryName, verbosity)
		fmt.Printf("[%s][%s] %s resource policies found.\n", cyan(m.output.CallingModule), cyan(m.AWSProfileStub), strconv.Itoa(len(m.output.Body)))
		//fmt.Printf("[%s][%s] Resource policies stored to: %s\n", cyan(m.output.CallingModule), cyan(m.AWSProfile), m.getLootDir())
	} else {
		fmt.Printf("[%s][%s] No resource policies found, skipping the creation of an output file.\n", cyan(m.output.CallingModule), cyan(m.AWSProfileStub))
	}
	fmt.Printf("[%s][%s] For context and next steps: https://github.com/BishopFox/cloudfox/wiki/AWS-Commands#%s\n", cyan(m.output.CallingModule), cyan(m.AWSProfileStub), m.output.CallingModule)
}

func (m *ResourceTrustsModule) executeChecks(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2, includeKms bool) {
	defer wg.Done()

	servicemap := &awsservicemap.AwsServiceMap{
		JsonFileSource: "DOWNLOAD_FROM_AWS",
	}
	res, err := servicemap.IsServiceInRegion("sns", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getSNSTopicsPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("sqs", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getSQSQueuesPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("ecr", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getECRRecordsPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("codebuild", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getCodeBuildResourcePoliciesPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("lambda", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getLambdaPolicyPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("efs", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getEFSfilesystemPoliciesPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("secretsmanager", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getSecretsManagerSecretsPoliciesPerRegion(r, wg, semaphore, dataReceiver)
	}

	res, err = servicemap.IsServiceInRegion("glue", r)
	if err != nil {
		m.modLog.Error(err)
	}
	if res {
		m.CommandCounter.Total++
		wg.Add(1)
		m.getGlueResourcePoliciesPerRegion(r, wg, semaphore, dataReceiver)
	}

	if includeKms && m.KMSClient != nil {
		res, err = servicemap.IsServiceInRegion("kms", r)
		if err != nil {
			m.modLog.Error(err)
		}
		if res {
			m.CommandCounter.Total++
			wg.Add(1)
			m.getKMSPoliciesPerRegion(r, wg, semaphore, dataReceiver)
		}
	}

	if m.APIGatewayClient != nil {
		res, err = servicemap.IsServiceInRegion("apigateway", r)
		if err != nil {
			m.modLog.Error(err)
		}
		if res {
			m.CommandCounter.Total++
			wg.Add(1)
			m.getAPIGatewayPoliciesPerRegion(r, wg, semaphore, dataReceiver)
		}
	}

	if m.EC2Client != nil {
		res, err = servicemap.IsServiceInRegion("ec2", r)
		if err != nil {
			m.modLog.Error(err)
		}
		if res {
			m.CommandCounter.Total++
			wg.Add(1)
			m.getVPCEndpointPoliciesPerRegion(r, wg, semaphore, dataReceiver)
		}
	}

	if m.OpenSearchClient != nil {
		res, err = servicemap.IsServiceInRegion("es", r)
		if err != nil {
			m.modLog.Error(err)
		}
		if res {
			m.CommandCounter.Total++
			wg.Add(1)
			m.getOpenSearchPoliciesPerRegion(r, wg, semaphore, dataReceiver)
		}
	}
}

func (m *ResourceTrustsModule) Receiver(receiver chan Resource2, receiverDone chan bool) {
	defer close(receiverDone)
	for {
		select {
		case data := <-receiver:
			m.Resources2 = append(m.Resources2, data)
		case <-receiverDone:
			receiverDone <- true
			return
		}
	}
}

func (m *ResourceTrustsModule) getSNSTopicsPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxSNSClient := InitCloudFoxSNSClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.WrapTable, m.AWSMFAToken)

	ListTopics, err := cloudFoxSNSClient.listTopics(r)
	if err != nil {
		m.modLog.Error(err.Error())
		return
	}

	for _, t := range ListTopics {
		var statementSummaryInEnglish string
		var isInteresting = "No"
		topic, err := cloudFoxSNSClient.getTopicWithAttributes(aws.ToString(t.TopicArn), r)
		if err != nil {
			m.modLog.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}
		parsedArn, err := arn.Parse(aws.ToString(t.TopicArn))
		if err != nil {
			topic.Name = aws.ToString(t.TopicArn)
		} else {
			topic.Name = parsedArn.Resource
			topic.Region = parsedArn.Region
		}

		// check if topic is public or not
		if topic.Policy.IsPublic() {
			topic.IsPublic = magenta("Yes")
			isInteresting = magenta("Yes")
		} else {
			topic.IsPublic = "No"
		}

		// If there is a resource policy, convert the resource policy into plain english
		if !topic.Policy.IsEmpty() {

			for i, statement := range topic.Policy.Statement {
				var prefix string = ""
				if len(topic.Policy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = statementSummaryInEnglish + prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}

			}
			statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
			if isResourcePolicyInteresting(statementSummaryInEnglish) {
				//magenta(statementSummaryInEnglish)
				isInteresting = magenta("Yes")
			}

			dataReceiver <- Resource2{
				AccountID:             aws.ToString(m.Caller.Account),
				ARN:                   aws.ToString(t.TopicArn),
				ResourcePolicySummary: statementSummaryInEnglish,
				Public:                topic.IsPublic,
				Region:                parsedArn.Region,
				Name:                  topic.Name,
				Interesting:           isInteresting,
			}

		}
	}

}

func (m *ResourceTrustsModule) getS3Buckets(wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxS3Client := initCloudFoxS3Client(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.AWSMFAToken)

	ListBuckets, err := sdk.CachedListBuckets(cloudFoxS3Client.S3Client, aws.ToString(m.Caller.Account))
	if err != nil {
		m.modLog.Error(err.Error())
		return
	}

	for _, b := range ListBuckets {
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		bucket := &BucketRow{
			Arn: fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(b.Name)),
		}
		name := aws.ToString(b.Name)
		region, err := sdk.CachedGetBucketLocation(cloudFoxS3Client.S3Client, aws.ToString(m.Caller.Account), name)
		if err != nil {
			m.modLog.Error(err.Error())
			continue
		}

		policyJSON, err := sdk.CachedGetBucketPolicy(cloudFoxS3Client.S3Client, aws.ToString(m.Caller.Account), region, aws.ToString(b.Name))
		if err != nil {
			m.modLog.Error(err.Error())
			continue
		} else {
			bucket.PolicyJSON = policyJSON
		}

		policy, err := policy.ParseJSONPolicy([]byte(policyJSON))
		if err != nil {
			m.modLog.Error(fmt.Sprintf("parsing bucket access policy (%s) as JSON: %s", name, err))
		} else {
			bucket.Policy = policy
		}
		// easier to just set the default state to be no and only flip it to yes if we have a case that matches
		if policy.IsPublic() {
			bucket.IsPublic = magenta("Yes")
			isInteresting = magenta("Yes")
		} else {
			bucket.IsPublic = "No"
		}

		// If there is a resource policy, convert the resource policy into plain english
		if !bucket.Policy.IsEmpty() {
			for i, statement := range bucket.Policy.Statement {
				var prefix string = ""
				if len(bucket.Policy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = statementSummaryInEnglish + prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
			}
			statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
			if isResourcePolicyInteresting(statementSummaryInEnglish) {
				//magenta(statementSummaryInEnglish)
				isInteresting = magenta("Yes")
			}

			dataReceiver <- Resource2{
				AccountID:             aws.ToString(m.Caller.Account),
				ARN:                   bucket.Arn,
				ResourcePolicySummary: statementSummaryInEnglish,
				Public:                bucket.IsPublic,
				Region:                region,
				Name:                  name,
				Interesting:           isInteresting,
			}
		}
	}
}

func (m *ResourceTrustsModule) getSQSQueuesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxSQSClient := InitSQSClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ListQueues, err := cloudFoxSQSClient.listQueues(r)
	if err != nil {
		m.modLog.Error(err.Error())
		return
	}

	for _, q := range ListQueues {
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		var isPublic string = "No"
		queue, err := cloudFoxSQSClient.getQueueWithAttributes(q, r)
		if err != nil {
			m.modLog.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if !queue.Policy.IsEmpty() {
			if queue.Policy.IsPublic() {
				isPublic = magenta("Yes")
				isInteresting = magenta("Yes")
			}
			for i, statement := range queue.Policy.Statement {
				prefix := ""
				if len(queue.Policy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(&queue.Arn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  q,
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

func (m *ResourceTrustsModule) getECRRecordsPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxECRClient := InitECRClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	DescribeRepositories, err := sdk.CachedECRDescribeRepositories(cloudFoxECRClient.ECRClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		m.modLog.Error(err.Error())
		return
	}

	for _, repo := range DescribeRepositories {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		repoPolicy, err := cloudFoxECRClient.getECRRepositoryPolicy(r, aws.ToString(repo.RepositoryName))
		if err != nil {
			m.modLog.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if !repoPolicy.IsEmpty() {
			if repoPolicy.IsPublic() {
				isPublic = magenta("Yes")
				isInteresting = magenta("Yes")
			} else {
				isPublic = "No"
			}
			for i, statement := range repoPolicy.Statement {
				prefix := ""
				if len(repoPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(repo.RepositoryArn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(repo.RepositoryName),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

func (m *ResourceTrustsModule) getCodeBuildResourcePoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxCodeBuildClient := InitCodeBuildClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ListProjects, err := sdk.CachedCodeBuildListProjects(cloudFoxCodeBuildClient.CodeBuildClient, aws.ToString(cloudFoxCodeBuildClient.Caller.Account), r)
	if err != nil {
		m.modLog.Error(err.Error())
		return
	}

	for _, p := range ListProjects {
		var projectPolicy policy.Policy
		var statementSummaryInEnglish string
		var isPublic string
		var isInteresting string = "No"
		project, err := sdk.CachedCodeBuildBatchGetProjects(cloudFoxCodeBuildClient.CodeBuildClient, aws.ToString(cloudFoxCodeBuildClient.Caller.Account), r, p)
		if err != nil {
			m.modLog.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		policyJSON, err := sdk.CachedCodeBuildGetResourcePolicy(cloudFoxCodeBuildClient.CodeBuildClient, aws.ToString(cloudFoxCodeBuildClient.Caller.Account), r, p)
		if err != nil {
			sharedLogger.Error(err.Error())
			continue
		}

		projectPolicy, err = policy.ParseJSONPolicy([]byte(policyJSON))

		if !projectPolicy.IsEmpty() {
			if projectPolicy.IsPublic() {
				isPublic = magenta("Yes")
				isInteresting = magenta("Yes")

			} else {
				isPublic = "No"
			}
			for i, statement := range projectPolicy.Statement {
				prefix := ""
				if len(projectPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(project.Arn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(project.Name),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

func (m *ResourceTrustsModule) getLambdaPolicyPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxLambdaClient := InitLambdaClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ListFunctions, err := cloudFoxLambdaClient.listFunctions(r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, f := range ListFunctions {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		functionPolicy, err := cloudFoxLambdaClient.getResourcePolicy(r, aws.ToString(f.FunctionName))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if !functionPolicy.IsEmpty() {
			if functionPolicy.IsPublic() {
				isPublic = magenta("Yes")
				isInteresting = magenta("Yes")

			} else {
				isPublic = "No"
			}
			for i, statement := range functionPolicy.Statement {
				prefix := ""
				if len(functionPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(f.FunctionArn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(f.FunctionName),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

func (m *ResourceTrustsModule) getEFSfilesystemPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxEFSClient := InitFileSystemsClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ListFileSystems, err := sdk.CachedDescribeFileSystems(cloudFoxEFSClient.EFSClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, fs := range ListFileSystems {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		fsPolicy, err := sdk.CachedDescribeFileSystemPolicy(cloudFoxEFSClient.EFSClient, aws.ToString(fs.FileSystemId), r, aws.ToString(m.Caller.Account))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if fsPolicy.IsPublic() {
			isPublic = magenta("Yes")
			isInteresting = magenta("Yes")

		} else {
			isPublic = "No"
		}

		if !fsPolicy.IsEmpty() {
			for i, statement := range fsPolicy.Statement {
				prefix := ""
				if len(fsPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(fs.FileSystemArn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(fs.Name),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

// getSecretsManagerSecretsPoliciesPerRegion retrieves the resource policies for all Secrets Manager secrets in the specified region.
// It sends the resulting Resource2 objects to the dataReceiver channel.
// It uses a semaphore to limit the number of concurrent requests and a WaitGroup to wait for all requests to complete.
// It takes the region to search in, the WaitGroup to use, the semaphore to use, and the dataReceiver channel to send results to.
func (m *ResourceTrustsModule) getSecretsManagerSecretsPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxSecretsManagerClient := InitSecretsManagerClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ListSecrets, err := sdk.CachedSecretsManagerListSecrets(cloudFoxSecretsManagerClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, s := range ListSecrets {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting string = "No"
		secretPolicy, err := sdk.CachedSecretsManagerGetResourcePolicy(cloudFoxSecretsManagerClient, aws.ToString(s.ARN), r, aws.ToString(m.Caller.Account))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if secretPolicy.IsPublic() {
			isPublic = magenta("Yes")
			isInteresting = magenta("Yes")

		} else {
			isPublic = "No"
		}

		if !secretPolicy.IsEmpty() {
			for i, statement := range secretPolicy.Statement {
				prefix := ""
				if len(secretPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}
				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(s.ARN),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(s.Name),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

// getKMSPoliciesPerRegion retrieves the resource policies for all KMS keys in a specified region.
// It sends the resulting Resource2 objects to the dataReceiver channel.
// It uses a semaphore to limit the number of concurrent requests and a WaitGroup to wait for all requests to complete.
// It takes the region to search in, the WaitGroup to use, the semaphore to use, and the dataReceiver channel to send results to.
func (m *ResourceTrustsModule) getKMSPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()
	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	listKeys, err := sdk.CachedKMSListKeys(*m.KMSClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, key := range listKeys {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting = "No"

		keyPolicy, err := sdk.CachedKMSGetKeyPolicy(*m.KMSClient, aws.ToString(m.Caller.Account), r, aws.ToString(key.KeyId))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if keyPolicy.IsPublic() {
			isPublic = magenta("Yes")
			isInteresting = magenta("Yes")
		} else {
			isPublic = "No"
		}

		if !keyPolicy.IsEmpty() {
			for i, statement := range keyPolicy.Statement {
				prefix := ""
				if len(keyPolicy.Statement) > 1 {
					prefix = fmt.Sprintf("Statement %d says: ", i)
					statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
				} else {
					statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				}

				statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
				if isResourcePolicyInteresting(statementSummaryInEnglish) {
					//magenta(statementSummaryInEnglish)
					isInteresting = magenta("Yes")
				}

				dataReceiver <- Resource2{
					AccountID:             aws.ToString(m.Caller.Account),
					ARN:                   aws.ToString(key.KeyArn),
					ResourcePolicySummary: statementSummaryInEnglish,
					Public:                isPublic,
					Name:                  aws.ToString(key.KeyId),
					Region:                r,
					Interesting:           isInteresting,
				}
			}
		}
	}
}

func (m *ResourceTrustsModule) getAPIGatewayPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()
	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	restAPIs, err := sdk.CachedApiGatewayGetRestAPIs(*m.APIGatewayClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, restAPI := range restAPIs {

		if sdk.IsPublicApiGateway(&restAPI) {
			continue
		}

		var isPublic = "No"
		var statementSummaryInEnglish string
		var isInteresting = "No"

		if restAPI.Policy != nil && *restAPI.Policy != "" {

			// remove backslashes from the policy JSON
			policyJson := strings.ReplaceAll(aws.ToString(restAPI.Policy), `\"`, `"`)

			restAPIPolicy, err := policy.ParseJSONPolicy([]byte(policyJson))
			if err != nil {
				sharedLogger.Error(fmt.Errorf("parsing policy (%s) as JSON: %s", aws.ToString(restAPI.Name), err))
				m.CommandCounter.Error++
				continue
			}

			if !restAPIPolicy.IsEmpty() {
				for i, statement := range restAPIPolicy.Statement {
					prefix := ""
					if len(restAPIPolicy.Statement) > 1 {
						prefix = fmt.Sprintf("Statement %d says: ", i)
						statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
					} else {
						statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
					}

					statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
					if isResourcePolicyInteresting(statementSummaryInEnglish) {
						//magenta(statementSummaryInEnglish)
						isInteresting = magenta("Yes")
					}

					dataReceiver <- Resource2{
						AccountID:             aws.ToString(m.Caller.Account),
						ARN:                   fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*", r, *m.Caller.Account, *restAPI.Id),
						ResourcePolicySummary: statementSummaryInEnglish,
						Public:                isPublic,
						Name:                  aws.ToString(restAPI.Name),
						Region:                r,
						Interesting:           isInteresting,
					}
				}
			}
		} else {
			dataReceiver <- Resource2{
				AccountID:             aws.ToString(m.Caller.Account),
				ARN:                   fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*", r, *m.Caller.Account, *restAPI.Id),
				ResourcePolicySummary: statementSummaryInEnglish,
				Public:                isPublic,
				Name:                  aws.ToString(restAPI.Name),
				Region:                r,
				Interesting:           isInteresting,
			}
		}
	}
}

func (m *ResourceTrustsModule) getVPCEndpointPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()
	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	vpcEndpoints, err := sdk.CachedEC2DescribeVpcEndpoints(*m.EC2Client, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, vpcEndpoint := range vpcEndpoints {
		var isPublic = "No"
		var statementSummaryInEnglish string
		var isInteresting = "No"

		if vpcEndpoint.PolicyDocument != nil && *vpcEndpoint.PolicyDocument != "" {
			vpcEndpointPolicyJson := aws.ToString(vpcEndpoint.PolicyDocument)
			vpcEndpointPolicy, err := policy.ParseJSONPolicy([]byte(vpcEndpointPolicyJson))
			if err != nil {
				sharedLogger.Error(fmt.Errorf("parsing policy (%s) as JSON: %s", aws.ToString(vpcEndpoint.VpcEndpointId), err))
				m.CommandCounter.Error++
				continue
			}

			if !vpcEndpointPolicy.IsEmpty() {
				for i, statement := range vpcEndpointPolicy.Statement {
					prefix := ""
					if len(vpcEndpointPolicy.Statement) > 1 {
						prefix = fmt.Sprintf("Statement %d says: ", i)
						statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
					} else {
						statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
					}

					statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
					if isResourcePolicyInteresting(statementSummaryInEnglish) {
						//magenta(statementSummaryInEnglish)
						isInteresting = magenta("Yes")
					}

					dataReceiver <- Resource2{
						AccountID:             aws.ToString(m.Caller.Account),
						ARN:                   fmt.Sprintf("arn:aws:ec2:%s:%s:vpc-endpoint/%s", r, aws.ToString(m.Caller.Account), aws.ToString(vpcEndpoint.VpcEndpointId)),
						ResourcePolicySummary: statementSummaryInEnglish,
						Public:                isPublic,
						Name:                  aws.ToString(vpcEndpoint.VpcEndpointId),
						Region:                r,
						Interesting:           isInteresting,
					}
				}
			}
		} else {
			dataReceiver <- Resource2{
				AccountID:             aws.ToString(m.Caller.Account),
				ARN:                   fmt.Sprintf("arn:aws:ec2:%s:%s:vpc-endpoint/%s", r, aws.ToString(m.Caller.Account), aws.ToString(vpcEndpoint.VpcEndpointId)),
				ResourcePolicySummary: statementSummaryInEnglish,
				Public:                isPublic,
				Name:                  aws.ToString(vpcEndpoint.VpcEndpointId),
				Region:                r,
				Interesting:           isInteresting,
			}
		}
	}
}

func (m *ResourceTrustsModule) getOpenSearchPoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()
	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	openSearchDomains, err := sdk.CachedOpenSearchListDomainNames(*m.OpenSearchClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}
	for _, openSearchDomain := range openSearchDomains {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting = "No"

		openSearchDomainConfig, err := sdk.CachedOpenSearchDescribeDomainConfig(*m.OpenSearchClient, aws.ToString(m.Caller.Account), r, aws.ToString(openSearchDomain.DomainName))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if aws.ToBool(openSearchDomainConfig.AdvancedSecurityOptions.Options.Enabled) {
			isPublic = "No"
		} else {
			isPublic = magenta("Yes")
			isInteresting = magenta("Yes")
		}

		openSearchDomainStatus, err := sdk.CachedOpenSearchDescribeDomain(*m.OpenSearchClient, aws.ToString(m.Caller.Account), r, aws.ToString(openSearchDomain.DomainName))
		if err != nil {
			sharedLogger.Error(err.Error())
			m.CommandCounter.Error++
			continue
		}

		if openSearchDomainStatus.AccessPolicies != nil && *openSearchDomainStatus.AccessPolicies != "" {

			// remove backslashes from the policy JSON
			policyJson := strings.ReplaceAll(aws.ToString(openSearchDomainStatus.AccessPolicies), `\"`, `"`)

			openSearchDomainPolicy, err := policy.ParseJSONPolicy([]byte(policyJson))
			if err != nil {
				sharedLogger.Error(fmt.Errorf("parsing policy (%s) as JSON: %s", aws.ToString(openSearchDomainStatus.ARN), err))
				m.CommandCounter.Error++
				continue
			}

			if !openSearchDomainPolicy.IsEmpty() {
				for i, statement := range openSearchDomainPolicy.Statement {
					prefix := ""
					if len(openSearchDomainPolicy.Statement) > 1 {
						prefix = fmt.Sprintf("Statement %d says: ", i)
						statementSummaryInEnglish = prefix + statement.GetStatementSummaryInEnglish(*m.Caller.Account) + "\n"
					} else {
						statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
					}

					statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
					if isResourcePolicyInteresting(statementSummaryInEnglish) {
						//magenta(statementSummaryInEnglish)
						isInteresting = magenta("Yes")
					}

					dataReceiver <- Resource2{
						AccountID:             aws.ToString(m.Caller.Account),
						ARN:                   aws.ToString(openSearchDomainStatus.ARN),
						ResourcePolicySummary: statementSummaryInEnglish,
						Public:                isPublic,
						Name:                  aws.ToString(openSearchDomain.DomainName),
						Region:                r,
						Interesting:           isInteresting,
					}
				}
			}
		} else {
			dataReceiver <- Resource2{
				AccountID:             aws.ToString(m.Caller.Account),
				ARN:                   aws.ToString(openSearchDomainStatus.ARN),
				ResourcePolicySummary: statementSummaryInEnglish,
				Public:                isPublic,
				Name:                  aws.ToString(openSearchDomain.DomainName),
				Region:                r,
				Interesting:           isInteresting,
			}
		}
	}
}

func (m *ResourceTrustsModule) getGlueResourcePoliciesPerRegion(r string, wg *sync.WaitGroup, semaphore chan struct{}, dataReceiver chan Resource2) {
	defer func() {
		m.CommandCounter.Executing--
		m.CommandCounter.Complete++
		wg.Done()

	}()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	cloudFoxGlueClient := InitGlueClient(m.Caller, m.AWSProfileProvided, m.CloudFoxVersion, m.Goroutines, m.AWSMFAToken)

	ResourcePolicies, err := sdk.CachedGlueGetResourcePolicies(cloudFoxGlueClient, aws.ToString(m.Caller.Account), r)
	if err != nil {
		sharedLogger.Error(err.Error())
		return
	}

	for _, resourcePolicy := range ResourcePolicies {
		var isPublic string
		var statementSummaryInEnglish string
		var isInteresting string = "No"

		if resourcePolicy.IsPublic() {
			isPublic = magenta("Yes")
			isInteresting = magenta("Yes")

		} else {
			isPublic = "No"
		}

		if !resourcePolicy.IsEmpty() {
			for _, statement := range resourcePolicy.Statement {
				statementSummaryInEnglish = statement.GetStatementSummaryInEnglish(*m.Caller.Account)
				resources := statement.Resource
				for _, resource := range resources {

					statementSummaryInEnglish = strings.TrimSuffix(statementSummaryInEnglish, "\n")
					if isResourcePolicyInteresting(statementSummaryInEnglish) {
						//magenta(statementSummaryInEnglish)
						isInteresting = magenta("Yes")
					}

					dataReceiver <- Resource2{
						AccountID:             aws.ToString(m.Caller.Account),
						ARN:                   resource,
						ResourcePolicySummary: statementSummaryInEnglish,
						Public:                isPublic,
						Name:                  resource,
						Region:                r,
						Interesting:           isInteresting,
					}
				}
			}
		}
	}
}

func isResourcePolicyInteresting(statementSummaryInEnglish string) bool {
	// check if the statement has any of the following items, but make sure the check is case-insensitive
	// if it does, then return true
	// if it doesn't, then return false

	if //(strings.Contains(strings.ToLower(statementSummaryInEnglish), "*") && !strings.Contains(strings.ToLower(statementSummaryInEnglish), "denied")) ||
	(strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("Everyone")) && !strings.Contains(strings.ToLower(statementSummaryInEnglish), "denied")) ||
		strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("aws:PrincipalOrgID")) ||
		strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("aws:PrincipalAccount")) ||
		strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("aws:PrincipalOrgPaths")) ||
		strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("role")) ||
		strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("root")) ||
		(strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("aws:arn")) && !strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("SourceArn"))) ||
		(strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("aws:arn")) && !strings.Contains(strings.ToLower(statementSummaryInEnglish), strings.ToLower("SourceAccount"))) {

		return true
	}
	return false
}
