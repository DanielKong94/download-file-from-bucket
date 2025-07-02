package providers

import (
	"fmt"
)

// NewProvider creates a new provider based on the given options
func NewProvider(opts ProviderOptions) (Provider, error) {
	switch opts.Type {
	case ProviderTypeS3, ProviderTypeDigitalOcean:
		return NewS3Provider(opts)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", opts.Type)
	}
}

// GetProviderOptions converts a config provider to provider options
func GetProviderOptions(providerType, region, endpoint, accessKey, secretKey, bucket string, options map[string]string) ProviderOptions {
	var pType ProviderType
	switch providerType {
	case "s3":
		pType = ProviderTypeS3
	case "digitalocean":
		pType = ProviderTypeDigitalOcean
	default:
		pType = ProviderType(providerType)
	}

	return ProviderOptions{
		Type:      pType,
		Region:    region,
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		Options:   options,
	}
} 