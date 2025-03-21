package internal

import (
	"bufio"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/ptr"
	"github.com/bishopfox/awsservicemap"
	"github.com/kyokomi/emoji"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	TxtLoggerName = "root"
	TxtLog        = TxtLogger()
	UtilsFs       = afero.NewOsFs()
	credsMap      = map[string]aws.Credentials{}
	ConfigMap     = map[string]aws.Config{}
)

type CloudFoxRunData struct {
	Profile        string
	AccountID      string
	OutputLocation string
}

func init() {
	gob.Register(aws.Config{})
	gob.Register(sts.GetCallerIdentityOutput{})
	gob.Register(CloudFoxRunData{})
}

func InitializeCloudFoxRunData(AWSProfile string, version string, AwsMfaToken string, AWSOutputDirectory string) (CloudFoxRunData, error) {
	var runData CloudFoxRunData

	cacheDirectory := filepath.Join(AWSOutputDirectory, "cached-data", "aws")
	filename := filepath.Join(cacheDirectory, fmt.Sprintf("CloudFoxRunData-%s.json", AWSProfile))
	if _, err := os.Stat(filename); err == nil {
		// unmarshall the data from the file into type CloudFoxRunData

		// Open the file (this is not actually needed if you use os.ReadFile, so you can skip this)
		file, err := os.Open(filename)
		if err != nil {
			return CloudFoxRunData{}, err
		}
		defer file.Close()

		// Read the file content
		jsonData, err := os.ReadFile(filename)
		if err != nil {
			return CloudFoxRunData{}, err
		}

		// Unmarshal jsonData into runData (make sure to pass a pointer to runData)
		err = json.Unmarshal(jsonData, &runData)
		if err != nil {
			return CloudFoxRunData{}, err
		}

		return runData, nil

	}

	CallerIdentity, err := AWSWhoami(AWSProfile, version, AwsMfaToken)
	if err != nil {
		return CloudFoxRunData{}, err
	}
	outputLocation := filepath.Join(AWSOutputDirectory, "cloudfox-output", "aws", fmt.Sprintf("%s-%s", AWSProfile, ptr.ToString(CallerIdentity.Account)))

	runData = CloudFoxRunData{
		Profile:        AWSProfile,
		AccountID:      aws.ToString(CallerIdentity.Account),
		OutputLocation: outputLocation,
	}

	// Marshall the data to a file
	err = os.MkdirAll(cacheDirectory, 0755)
	if err != nil {
		return CloudFoxRunData{}, err
	}
	file, err := os.Create(filename)
	if err != nil {
		return CloudFoxRunData{}, err
	}
	defer file.Close()
	jsonData, err := json.Marshal(runData)
	if err != nil {
		return CloudFoxRunData{}, err
	}
	_, err = file.Write(jsonData)
	if err != nil {
		return CloudFoxRunData{}, err
	}

	return runData, nil
}

func AWSConfigFileLoader(AWSProfile string, version string, AwsMfaToken string) aws.Config {
	// Loads the AWS config file and returns a config object

	var cfg aws.Config
	var err error
	// cacheKey := fmt.Sprintf("AWSConfigFileLoader-%s", AWSProfile)
	// cached, found := Cache.Get(cacheKey)
	// if found {
	// 	cfg = cached.(aws.Config)
	// 	return cfg
	// }

	// Check if the profile is already in the config map. If not, load it and retrieve the credentials. If it is, return the cached config object
	// The AssumeRoleOptions below are used to pass the MFA token to the AssumeRole call (when applicable)
	if _, ok := ConfigMap[AWSProfile]; !ok {
		// Ensures the profile in the aws config file meets all requirements (valid keys and a region defined). I noticed some calls fail without a default region.
		if AwsMfaToken != "" {
			cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(AWSProfile), config.WithDefaultRegion("us-east-1"), config.WithRetryer(
				func() aws.Retryer {
					return retry.AddWithMaxAttempts(retry.NewStandard(), 3)
				}), config.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
				options.TokenProvider = func() (string, error) {
					return AwsMfaToken, nil
				}
			}),
			)
		} else {
			cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(AWSProfile), config.WithDefaultRegion("us-east-1"), config.WithRetryer(
				func() aws.Retryer {
					return retry.AddWithMaxAttempts(retry.NewStandard(), 3)
				}), config.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
				options.TokenProvider = stscreds.StdinTokenProvider
			}),
			)
		}

		if err != nil {
			//fmt.Println(err)
			if AWSProfile != "" {
				TxtLog.Println(err)
				fmt.Printf("[%s][%s] The specified profile [%s] does not exist or there was an error loading the credentials.\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)), cyan(AWSProfile), AWSProfile)
				TxtLog.Fatalf("Could not retrieve the specified profile name %s", err)
			} else {
				fmt.Printf("[%s][%s] Error retrieving credentials from environment variables, or the instance metadata service.\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)), cyan(AWSProfile))
				TxtLog.Fatalf("[%s][%s]Error retrieving credentials from environment variables, or the instance metadata service.\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)), cyan(AWSProfile))
			}
			//os.Exit(1)
		}

		_, err := cfg.Credentials.Retrieve(context.TODO())

		if err != nil {
			fmt.Printf("[%s][%s] Error retrieving credentials from environment variables, or the instance metadata service.\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)), cyan(AWSProfile))

		} else {
			// update the config map with the new config for future lookups
			ConfigMap[AWSProfile] = cfg
			//return the config object for this first iteration
			//Cache.Set(cacheKey, cfg, cache.DefaultExpiration)
			return cfg

		}
	} else {
		//fmt.Println("Using cached config")
		cfg = ConfigMap[AWSProfile]
		return cfg
	}
	//Cache.Set(cacheKey, cfg, cache.DefaultExpiration)
	return cfg
}

func AWSWhoami(awsProfile string, version string, AwsMfaToken string) (*sts.GetCallerIdentityOutput, error) {

	cacheKey := fmt.Sprintf("sts-getCallerIdentity-%s", awsProfile)
	if cached, found := Cache.Get(cacheKey); found {
		// Correct type assertion: assert the type, not a variable.
		if cachedValue, ok := cached.(*sts.GetCallerIdentityOutput); ok {
			return cachedValue, nil
		}
		// Handle the case where type assertion fails, if necessary.
	}

	// Connects to STS and checks caller identity. Same as running "aws sts get-caller-identity"
	//fmt.Printf("[%s] Retrieving caller's identity\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)))
	STSService := sts.NewFromConfig(AWSConfigFileLoader(awsProfile, version, AwsMfaToken))
	CallerIdentity, err := STSService.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Printf("[%s][%s] Could not get caller's identity\n\nError: %s\n\n", cyan(emoji.Sprintf(":fox:cloudfox v%s :fox:", version)), cyan(awsProfile), err)
		TxtLog.Printf("Could not get caller's identity: %s", err)
		return CallerIdentity, err

	}
	// Convert CallerIdentity to something i can store using the cache
	Cache.Set(cacheKey, CallerIdentity, cache.DefaultExpiration)
	return CallerIdentity, err
}

func GetEnabledRegions(awsProfile string, version string, AwsMfaToken string) []string {
	cacheKey := fmt.Sprintf("GetEnabledRegions-%s", awsProfile)
	cached, found := Cache.Get(cacheKey)
	if found {
		return cached.([]string)
	}

	var enabledRegions []string
	ec2Client := ec2.NewFromConfig(ConfigMap[awsProfile])
	regions, err := ec2Client.DescribeRegions(
		context.TODO(),
		&ec2.DescribeRegionsInput{
			AllRegions: aws.Bool(false),
		},
	)

	if err != nil {
		servicemap := &awsservicemap.AwsServiceMap{
			JsonFileSource: "DOWNLOAD_FROM_AWS",
		}
		AWSRegions, err := servicemap.GetAllRegions()
		if err != nil {
			TxtLog.Println(err)
		}
		return AWSRegions
	}

	for _, region := range regions.Regions {
		enabledRegions = append(enabledRegions, *region.RegionName)
	}
	Cache.Set(cacheKey, enabledRegions, cache.DefaultExpiration)
	return enabledRegions

}

// txtLogger - Returns the txt logger
func TxtLogger() *logrus.Logger {
	var txtFile *os.File
	var err error
	txtLogger := logrus.New()
	txtFile, err = os.OpenFile(fmt.Sprintf("%s/cloudfox-error.log", ptr.ToString(GetLogDirPath())), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		txtFile, err = os.OpenFile(fmt.Sprintf("./cloudfox-error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.InfoLevel)
	//txtLogger.SetReportCaller(true)

	return txtLogger
}

func CheckErr(e error, msg string) {
	if e != nil {
		TxtLog.Printf("[-] Error %s", msg)
	}
}

func GetAllAWSProfiles(AWSConfirm bool) []string {
	var AWSProfiles []string

	credentialsFile, err := UtilsFs.Open(config.DefaultSharedCredentialsFilename())
	CheckErr(err, "could not open default AWS credentials file")
	if err == nil {
		defer credentialsFile.Close()
		scanner := bufio.NewScanner(credentialsFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
				text = strings.TrimPrefix(text, "[")
				text = strings.TrimSuffix(text, "]")
				if !Contains(text, AWSProfiles) {
					AWSProfiles = append(AWSProfiles, text)
				}
			}
		}
	}

	configFile, err := UtilsFs.Open(config.DefaultSharedConfigFilename())
	CheckErr(err, "could not open default AWS credentials file")
	if err == nil {
		defer configFile.Close()
		scanner2 := bufio.NewScanner(configFile)
		scanner2.Split(bufio.ScanLines)
		for scanner2.Scan() {
			text := strings.TrimSpace(scanner2.Text())
			if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
				text = strings.TrimPrefix(text, "[profile ")
				text = strings.TrimPrefix(text, "[")
				text = strings.TrimSuffix(text, "]")
				if !Contains(text, AWSProfiles) {
					AWSProfiles = append(AWSProfiles, text)
				}
			}
		}
	}

	if !AWSConfirm {
		result := ConfirmSelectedProfiles(AWSProfiles)
		if !result {
			os.Exit(1)
		}
	}
	return AWSProfiles

}

func ConfirmSelectedProfiles(AWSProfiles []string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("[ %s] Identified profiles:\n\n", cyan(emoji.Sprintf(":fox:cloudfox :fox:")))
	for _, profile := range AWSProfiles {
		fmt.Printf("\t* %s\n", profile)
	}
	fmt.Printf("\n[ %s] Are you sure you'd like to run this command against the [%d] listed profile(s)? (Y\\n): ", cyan(emoji.Sprintf(":fox:cloudfox :fox:")), len(AWSProfiles))
	text, _ := reader.ReadString('\n')
	switch text {
	case "\n", "Y\n", "y\n":
		return true
	}
	return false

}

func GetSelectedAWSProfiles(AWSProfilesListPath string) []string {
	AWSProfilesListFile, err := UtilsFs.Open(AWSProfilesListPath)
	CheckErr(err, fmt.Sprintf("could not open given file %s", AWSProfilesListPath))
	if err != nil {
		fmt.Printf("\nError loading profiles. Could not open file at location[%s]\n", AWSProfilesListPath)
		os.Exit(1)
	}
	defer AWSProfilesListFile.Close()
	var AWSProfiles []string
	scanner := bufio.NewScanner(AWSProfilesListFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		profile := strings.TrimSpace(scanner.Text())
		if len(profile) != 0 {
			AWSProfiles = append(AWSProfiles, profile)
		}
	}
	return AWSProfiles
}

func removeBadPathChars(receivedPath *string) string {
	var path string
	var bannedPathChars *regexp.Regexp = regexp.MustCompile(`[<>:"'|?*]`)
	path = bannedPathChars.ReplaceAllString(aws.ToString(receivedPath), "_")

	return path

}

func BuildAWSPath(Caller sts.GetCallerIdentityOutput) string {
	var callerAccount = removeBadPathChars(Caller.Account)
	var callerUserID = removeBadPathChars(Caller.UserId)

	return fmt.Sprintf("%s-%s", callerAccount, callerUserID)
}

// this is all for the spinner and command counter
const clearln = "\r\x1b[2K"

type CommandCounter struct {
	Total     int
	Pending   int
	Complete  int
	Error     int
	Executing int
}

func SpinUntil(callingModuleName string, counter *CommandCounter, done chan bool, spinType string) {
	defer close(done)
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf(clearln+"[%s] Status: %d/%d %s complete (%d errors -- For details check %s)", cyan(callingModuleName), counter.Complete, counter.Total, spinType, counter.Error, fmt.Sprintf("%s/cloudfox-error.log", ptr.ToString(GetLogDirPath())))
		case <-done:
			fmt.Printf(clearln+"[%s] Status: %d/%d %s complete (%d errors -- For details check %s)\n", cyan(callingModuleName), counter.Complete, counter.Complete, spinType, counter.Error, fmt.Sprintf("%s/cloudfox-error.log", ptr.ToString(GetLogDirPath())))
			done <- true
			return
		}
	}
}

func ReorganizeAWSProfiles(allProfiles []string, mgmtProfile string) []string {
	// take the mgmt profile, move it from its current position to the front of the list
	var newProfiles []string
	newProfiles = append(newProfiles, mgmtProfile)
	for _, profile := range allProfiles {
		if profile != mgmtProfile {
			newProfiles = append(newProfiles, profile)
		}
	}
	return newProfiles
}
