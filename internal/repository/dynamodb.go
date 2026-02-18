package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	appconfig "github.com/losts/syun-eng/backend/internal/config"
)

// NewDynamoDBClient creates a new DynamoDB client
func NewDynamoDBClient(ctx context.Context, cfg *appconfig.Config) (*dynamodb.Client, error) {
	var awsCfg aws.Config
	var err error

	if cfg.DynamoDBEndpoint != "" {
		// Local development with DynamoDB Local
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.AWSRegion),
			config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(
					func(service, region string, options ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{
							URL:           cfg.DynamoDBEndpoint,
							SigningRegion: cfg.AWSRegion,
						}, nil
					},
				),
			),
		)
	} else {
		// Production
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.AWSRegion),
		)
	}

	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(awsCfg), nil
}
