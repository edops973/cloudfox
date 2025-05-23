package sdk

import (
	"context"
	"encoding/gob"
	"fmt"

	"github.com/BishopFox/cloudfox/internal"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	apiGatewayTypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/patrickmn/go-cache"
)

type APIGatewayClientInterface interface {
	GetRestApis(context.Context, *apigateway.GetRestApisInput, ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error)
	GetStages(context.Context, *apigateway.GetStagesInput, ...func(*apigateway.Options)) (*apigateway.GetStagesOutput, error)
	GetResources(context.Context, *apigateway.GetResourcesInput, ...func(*apigateway.Options)) (*apigateway.GetResourcesOutput, error)
	GetDomainNames(context.Context, *apigateway.GetDomainNamesInput, ...func(*apigateway.Options)) (*apigateway.GetDomainNamesOutput, error)
	GetBasePathMappings(context.Context, *apigateway.GetBasePathMappingsInput, ...func(*apigateway.Options)) (*apigateway.GetBasePathMappingsOutput, error)
	GetMethod(context.Context, *apigateway.GetMethodInput, ...func(*apigateway.Options)) (*apigateway.GetMethodOutput, error)
	GetUsagePlans(context.Context, *apigateway.GetUsagePlansInput, ...func(*apigateway.Options)) (*apigateway.GetUsagePlansOutput, error)
	GetUsagePlanKeys(context.Context, *apigateway.GetUsagePlanKeysInput, ...func(*apigateway.Options)) (*apigateway.GetUsagePlanKeysOutput, error)
}

type CachedGetMethodOutput struct {
	ApiKeyRequired      *bool
	AuthorizationScopes []string
	AuthorizationType   *string
	AuthorizerId        *string
	HttpMethod          *string
	MethodIntegration   *apiGatewayTypes.Integration
	MethodResponses     map[string]apiGatewayTypes.MethodResponse
	OperationName       *string
	RequestModels       map[string]string
	RequestParameters   map[string]bool
	RequestValidatorId  *string
	ResourceId          *string
}

type CachedGetStagesOutput struct {
	Item []apiGatewayTypes.Stage
}

func init() {
	gob.Register([]apiGatewayTypes.RestApi{})
	gob.Register(CachedGetStagesOutput{})
	gob.Register([]apiGatewayTypes.Resource{})
	gob.Register([]apiGatewayTypes.DomainName{})
	gob.Register([]apiGatewayTypes.BasePathMapping{})
	gob.Register(CachedGetMethodOutput{})
	gob.Register([]apiGatewayTypes.UsagePlan{})
	gob.Register([]apiGatewayTypes.UsagePlanKey{})
}

func IsPublicApiGateway(ra *apiGatewayTypes.RestApi) bool {
	for _, endpointType := range ra.EndpointConfiguration.Types {
		if endpointType == apiGatewayTypes.EndpointTypeRegional || endpointType == apiGatewayTypes.EndpointTypeEdge {
			return true
		}
	}

	return false
}

// CachedApiGatewayGetRestAPIs function that accepts a client, account id, region. Make sure it handles caching, the region option and pagination
func CachedApiGatewayGetRestAPIs(client APIGatewayClientInterface, accountID string, region string) ([]apiGatewayTypes.RestApi, error) {
	var PaginationControl *string
	var restAPIs []apiGatewayTypes.RestApi
	cacheKey := fmt.Sprintf("%s-apigateway-GetRestApis-%s", accountID, region)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.RestApi), nil
	}

	for {
		GetRestApis, err := client.GetRestApis(
			context.TODO(),
			&apigateway.GetRestApisInput{
				Position: PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return restAPIs, err
		}

		restAPIs = append(restAPIs, GetRestApis.Items...)

		//pagination
		if GetRestApis.Position == nil {
			break
		}
		PaginationControl = GetRestApis.Position
	}
	internal.Cache.Set(cacheKey, restAPIs, cache.DefaultExpiration)
	return restAPIs, nil
}

// create a CachedApiGatewayGetStages function that accepts a client, account id, region, and rest api id. Make sure it handles caching, the region option and pagination
func CachedApiGatewayGetStages(client APIGatewayClientInterface, accountID string, region string, restAPIID string) (*apigateway.GetStagesOutput, error) {
	cacheKey := fmt.Sprintf("%s-apigateway-GetStages-%s-%s", accountID, region, restAPIID)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		// Convert cached data back to GetStagesOutput before returning
		cachedOutput := cached.(*CachedGetStagesOutput) // Ensure this type assertion matches your caching logic
		return fromCachedGetStagesOutput(cachedOutput), nil
	}

	GetStages, err := client.GetStages(
		context.TODO(),
		&apigateway.GetStagesInput{
			RestApiId: &restAPIID,
		},
		func(o *apigateway.Options) {
			o.Region = region
		},
	)

	if err != nil {
		return &apigateway.GetStagesOutput{}, err
	}

	// Convert GetStagesOutput to CachedGetStagesOutput before caching
	cachedVersion := toCachedGetStagesOutput(GetStages)
	internal.Cache.Set(cacheKey, cachedVersion, cache.DefaultExpiration)
	return GetStages, err
}

// Convert from AWS SDK type to custom type for caching
func toCachedGetStagesOutput(gso *apigateway.GetStagesOutput) *CachedGetStagesOutput {
	return &CachedGetStagesOutput{
		Item: gso.Item,
	}
}

// Convert back to AWS SDK type after fetching from cache
func fromCachedGetStagesOutput(cgso *CachedGetStagesOutput) *apigateway.GetStagesOutput {
	return &apigateway.GetStagesOutput{
		Item: cgso.Item,
		// Initialize ResultMetadata or leave it as zero value if it's not required for your use case
	}
}

// create a CachedApiGatewayGetResources function that accepts a client, account id, region, and rest api id. Make sure it handles caching, the region option and pagination
func CachedApiGatewayGetResources(client APIGatewayClientInterface, accountID string, region string, restAPIID string) ([]apiGatewayTypes.Resource, error) {
	var PaginationControl *string
	var resources []apiGatewayTypes.Resource
	cacheKey := fmt.Sprintf("%s-apigateway-GetResources-%s-%s", accountID, region, restAPIID)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.Resource), nil
	}

	for {
		GetResources, err := client.GetResources(
			context.TODO(),
			&apigateway.GetResourcesInput{
				RestApiId: &restAPIID,
				Position:  PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return resources, err
		}

		resources = append(resources, GetResources.Items...)

		//pagination
		if GetResources.Position == nil {
			break
		}
		PaginationControl = GetResources.Position
	}
	internal.Cache.Set(cacheKey, resources, cache.DefaultExpiration)
	return resources, nil
}

// create a CachedApiGatewayGetDomainNames function that accepts a client, account id, region. Make sure it handles caching, the region option and pagination if needed
func CachedApiGatewayGetDomainNames(client APIGatewayClientInterface, accountID string, region string) ([]apiGatewayTypes.DomainName, error) {
	var PaginationControl *string
	var domainNames []apiGatewayTypes.DomainName
	cacheKey := fmt.Sprintf("%s-apigateway-GetDomainNames-%s", accountID, region)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.DomainName), nil
	}

	for {
		GetDomainNames, err := client.GetDomainNames(
			context.TODO(),
			&apigateway.GetDomainNamesInput{
				Position: PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return domainNames, err
		}

		domainNames = append(domainNames, GetDomainNames.Items...)

		//pagination
		if GetDomainNames.Position == nil {
			break
		}
		PaginationControl = GetDomainNames.Position
	}
	internal.Cache.Set(cacheKey, domainNames, cache.DefaultExpiration)
	return domainNames, nil
}

// create a CachedApiGatewayGetBasePathMappings function that accepts a client, account id, region. Make sure it handles caching, the region option and pagination if needed
func CachedApiGatewayGetBasePathMappings(client APIGatewayClientInterface, accountID string, region string, domain *string) ([]apiGatewayTypes.BasePathMapping, error) {
	var PaginationControl *string
	var basePathMappings []apiGatewayTypes.BasePathMapping
	cacheKey := fmt.Sprintf("%s-apigateway-GetBasePathMappings-%s-%s", accountID, region, aws.ToString(domain))
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.BasePathMapping), nil
	}

	for {
		GetBasePathMappings, err := client.GetBasePathMappings(
			context.TODO(),
			&apigateway.GetBasePathMappingsInput{
				DomainName: domain,
				Position:   PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return basePathMappings, err
		}

		basePathMappings = append(basePathMappings, GetBasePathMappings.Items...)

		//pagination
		if GetBasePathMappings.Position == nil {
			break
		}
		PaginationControl = GetBasePathMappings.Position
	}
	internal.Cache.Set(cacheKey, basePathMappings, cache.DefaultExpiration)
	return basePathMappings, nil
}

// create a CachedApiGatewayGetMethod function that accepts a client, account id, region, rest api id, and resource id. Make sure it handles caching, the region option and pagination if needed
func CachedApiGatewayGetMethod(client APIGatewayClientInterface, accountID string, region string, restAPIID string, resourceID string, method string) (*apigateway.GetMethodOutput, error) {

	cacheKey := fmt.Sprintf("%s-apigateway-GetMethod-%s-%s-%s-%s", accountID, region, restAPIID, resourceID, method)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		// Convert cached data back to GetMethodOutput before returning
		cachedOutput := cached.(*CachedGetMethodOutput) // Ensure this type assertion matches your caching logic
		return fromCachedGetMethodOutput(cachedOutput), nil
	}

	GetMethod, err := client.GetMethod(
		context.TODO(),
		&apigateway.GetMethodInput{
			RestApiId:  &restAPIID,
			ResourceId: &resourceID,
			HttpMethod: &method,
		},
		func(o *apigateway.Options) {
			o.Region = region
		},
	)

	if err != nil {
		return &apigateway.GetMethodOutput{}, err
	}

	// Convert GetMethodOutput to CachedGetMethodOutput before caching
	cachedVersion := toCachedGetMethodOutput(GetMethod)
	internal.Cache.Set(cacheKey, cachedVersion, cache.DefaultExpiration)
	return GetMethod, nil

}

// Convert from AWS SDK type to custom type for caching
func toCachedGetMethodOutput(gmo *apigateway.GetMethodOutput) *CachedGetMethodOutput {
	return &CachedGetMethodOutput{
		ApiKeyRequired:      gmo.ApiKeyRequired,
		AuthorizationScopes: gmo.AuthorizationScopes,
		AuthorizationType:   gmo.AuthorizationType,
		AuthorizerId:        gmo.AuthorizerId,
		HttpMethod:          gmo.HttpMethod,
		MethodIntegration:   gmo.MethodIntegration,
		MethodResponses:     gmo.MethodResponses,
		OperationName:       gmo.OperationName,
		RequestModels:       gmo.RequestModels,
		RequestParameters:   gmo.RequestParameters,
		RequestValidatorId:  gmo.RequestValidatorId,
	}
}

// Convert back to AWS SDK type after fetching from cache
func fromCachedGetMethodOutput(cgmo *CachedGetMethodOutput) *apigateway.GetMethodOutput {
	return &apigateway.GetMethodOutput{
		ApiKeyRequired:      cgmo.ApiKeyRequired,
		AuthorizationScopes: cgmo.AuthorizationScopes,
		AuthorizationType:   cgmo.AuthorizationType,
		AuthorizerId:        cgmo.AuthorizerId,
		HttpMethod:          cgmo.HttpMethod,
		MethodIntegration:   cgmo.MethodIntegration,
		MethodResponses:     cgmo.MethodResponses,
		OperationName:       cgmo.OperationName,
		RequestModels:       cgmo.RequestModels,
		RequestParameters:   cgmo.RequestParameters,
		RequestValidatorId:  cgmo.RequestValidatorId,
		// ResultMetadata: This will be missing or zero value; handle accordingly
	}
}

// create a CachedApiGatewayGetUsagePlans function that accepts a client, account id, region. Make sure it handles caching, the region option and pagination if needed
func CachedApiGatewayGetUsagePlans(client APIGatewayClientInterface, accountID string, region string) ([]apiGatewayTypes.UsagePlan, error) {
	var PaginationControl *string
	var usagePlans []apiGatewayTypes.UsagePlan
	cacheKey := fmt.Sprintf("%s-apigateway-GetUsagePlans-%s", accountID, region)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.UsagePlan), nil
	}

	for {
		GetUsagePlans, err := client.GetUsagePlans(
			context.TODO(),
			&apigateway.GetUsagePlansInput{
				Position: PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return usagePlans, err
		}

		usagePlans = append(usagePlans, GetUsagePlans.Items...)

		//pagination
		if GetUsagePlans.Position == nil {
			break
		}
		PaginationControl = GetUsagePlans.Position
	}

	internal.Cache.Set(cacheKey, usagePlans, cache.DefaultExpiration)
	return usagePlans, nil
}

// create a CachedApiGatewayGetUsagePlanKeys function that accepts a client, account id, region, and usage plan id. Make sure it handles caching, the region option and pagination if needed
func CachedApiGatewayGetUsagePlanKeys(client APIGatewayClientInterface, accountID string, region string, usagePlanID string) ([]apiGatewayTypes.UsagePlanKey, error) {
	var PaginationControl *string
	var usagePlanKeys []apiGatewayTypes.UsagePlanKey
	cacheKey := fmt.Sprintf("%s-apigateway-GetUsagePlanKeys-%s-%s", accountID, region, usagePlanID)
	cached, found := internal.Cache.Get(cacheKey)
	if found {
		return cached.([]apiGatewayTypes.UsagePlanKey), nil
	}

	for {
		GetUsagePlanKeys, err := client.GetUsagePlanKeys(
			context.TODO(),
			&apigateway.GetUsagePlanKeysInput{
				UsagePlanId: &usagePlanID,
				Position:    PaginationControl,
			},
			func(o *apigateway.Options) {
				o.Region = region
			},
		)

		if err != nil {
			return usagePlanKeys, err
		}

		usagePlanKeys = append(usagePlanKeys, GetUsagePlanKeys.Items...)

		//pagination
		if GetUsagePlanKeys.Position == nil {
			break
		}
		PaginationControl = GetUsagePlanKeys.Position
	}

	internal.Cache.Set(cacheKey, usagePlanKeys, cache.DefaultExpiration)
	return usagePlanKeys, nil
}
