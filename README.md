# Download File from Bucket

A fast and flexible Go application to clone entire folders from cloud storage buckets with support for multiple providers.

## Features

- **Multi-Provider Support**: AWS S3, DigitalOcean Spaces, and any S3-compatible service
- **Concurrent Downloads**: Configurable concurrency for fast downloads
- **Flexible Configuration**: Configuration file, environment variables, or CLI flags
- **Progress Tracking**: Real-time progress updates during downloads
- **Extensible Architecture**: Easy to add support for new cloud providers

## Supported Providers

- **AWS S3**: Native support with automatic credential detection
- **DigitalOcean Spaces**: Full compatibility with S3-compatible API
- **Custom S3-Compatible Services**: Configurable endpoints for other providers

## Installation

### From Source

```bash
git clone <repository-url>
cd download-file-from-bucket
go build -o download-bucket .
```

### Install Dependencies

```bash
go mod tidy
```

## Configuration

### 1. Configuration File

Create a config file at `~/.download-bucket/config.yaml`:

```yaml
providers:
  aws:
    type: s3
    region: us-west-2
    access_key: YOUR_AWS_ACCESS_KEY
    secret_key: YOUR_AWS_SECRET_KEY
    bucket: my-default-bucket
  
  digitalocean:
    type: digitalocean
    region: nyc3
    endpoint: https://nyc3.digitaloceanspaces.com
    access_key: YOUR_DO_ACCESS_KEY
    secret_key: YOUR_DO_SECRET_KEY
    bucket: my-default-space
```

### 2. Environment Variables

```bash
# AWS S3
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-west-2
export AWS_BUCKET=my-bucket

# DigitalOcean Spaces
export DO_ACCESS_KEY_ID=your_access_key
export DO_SECRET_ACCESS_KEY=your_secret_key
export DO_REGION=nyc3
export DO_BUCKET=my-space
```

### 3. CLI Configuration

```bash
# Set up AWS provider
./download-bucket config set aws \
  --access-key=YOUR_ACCESS_KEY \
  --secret-key=YOUR_SECRET_KEY \
  --region=us-west-2 \
  --bucket=my-bucket

# Set up DigitalOcean provider
./download-bucket config set digitalocean \
  --access-key=YOUR_ACCESS_KEY \
  --secret-key=YOUR_SECRET_KEY \
  --region=nyc3 \
  --bucket=my-space
```

## Usage

### Basic Usage

```bash
# Clone from S3
./download-bucket clone s3://my-bucket/folder/ ./local-folder

# Clone from DigitalOcean Spaces
./download-bucket clone spaces://my-space/data/ ./local-data

# Clone with specific provider
./download-bucket clone --provider=aws s3://my-bucket/images/ ./images
```

### Advanced Usage

```bash
# High concurrency download
./download-bucket clone --concurrency=10 s3://my-bucket/large-folder/ ./local

# Verbose output
./download-bucket clone --verbose s3://my-bucket/folder/ ./local

# Override credentials
./download-bucket clone \
  --access-key=TEMP_KEY \
  --secret-key=TEMP_SECRET \
  --region=eu-west-1 \
  s3://eu-bucket/data/ ./data

# Custom endpoint (for other S3-compatible services)
./download-bucket clone \
  --endpoint=https://custom-s3.example.com \
  --access-key=KEY \
  --secret-key=SECRET \
  s3://custom-bucket/folder/ ./local
```

### URL Formats

The application supports multiple URL formats:

1. **S3 URLs**: `s3://bucket-name/path/to/folder/`
2. **Spaces URLs**: `spaces://space-name/path/to/folder/`
3. **Full DigitalOcean URLs**: `https://region.digitaloceanspaces.com/space-name/path/`

### Configuration Management

```bash
# List configured providers
./download-bucket config list

# Set up a new provider
./download-bucket config set my-provider \
  --type=s3 \
  --access-key=KEY \
  --secret-key=SECRET \
  --region=us-east-1 \
  --endpoint=https://custom-endpoint.com \
  --bucket=default-bucket
```

## CLI Reference

### Global Flags

- `--verbose`: Enable verbose output
- `--config`: Specify custom config file path

### Clone Command

```bash
./download-bucket clone [flags] <source> <destination>
```

#### Flags

- `--provider`: Specify cloud provider (aws, digitalocean)
- `--concurrency`: Number of concurrent downloads (default: 5)
- `--access-key`: Access key (overrides config)
- `--secret-key`: Secret key (overrides config)
- `--region`: Region (overrides config)
- `--endpoint`: Custom endpoint (overrides config)
- `--bucket`: Bucket name (overrides URL)

### Config Commands

```bash
# Set provider configuration
./download-bucket config set <provider-name> [flags]

# List configured providers
./download-bucket config list
```

## Examples

### AWS S3

```bash
# Using environment variables
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
./download-bucket clone s3://my-bucket/photos/ ./photos

# Using CLI flags
./download-bucket clone \
  --access-key=your_key \
  --secret-key=your_secret \
  --region=us-west-2 \
  s3://my-bucket/documents/ ./documents
```

### DigitalOcean Spaces

```bash
# Using spaces:// URL
./download-bucket clone \
  --access-key=your_key \
  --secret-key=your_secret \
  spaces://my-space/backups/ ./backups

# Using full HTTPS URL
./download-bucket clone \
  --access-key=your_key \
  --secret-key=your_secret \
  https://nyc3.digitaloceanspaces.com/my-space/data/ ./data
```

### Custom S3-Compatible Service

```bash
./download-bucket clone \
  --endpoint=https://minio.example.com \
  --access-key=minio_key \
  --secret-key=minio_secret \
  s3://my-bucket/files/ ./files
```

## Architecture

The application follows a modular architecture:

- **Providers**: Pluggable cloud storage providers (currently S3-compatible)
- **Downloader**: Handles concurrent downloading with progress tracking
- **Config**: Flexible configuration management
- **CLI**: User-friendly command-line interface

### Adding New Providers

To add a new cloud provider:

1. Implement the `Provider` interface in the `providers` package
2. Add the provider type to `ProviderType` constants
3. Update the factory function in `providers/factory.go`
4. Add configuration handling in the config package

## Error Handling

The application provides detailed error messages and handles:

- Network failures with automatic retries
- Authentication errors
- Invalid URLs or configurations
- File system errors
- Partial downloads

## Performance

- **Concurrent Downloads**: Configurable concurrency level
- **Memory Efficient**: Streams files directly to disk
- **Progress Tracking**: Real-time download progress
- **Resumable Downloads**: Handles interrupted downloads gracefully

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License. 