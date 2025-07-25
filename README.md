# Download File from Bucket

A fast and flexible Go application to clone entire folders from cloud storage buckets with support for multiple providers.

## Features

- **Multi-Provider Support**: AWS S3, DigitalOcean Spaces, and any S3-compatible service
- **Concurrent Downloads**: Configurable concurrency for fast downloads
- **Enhanced UI**: Beautiful progress bars, real-time statistics, and colorful output
- **Flexible Configuration**: Configuration file, environment variables, or CLI flags
- **Progress Tracking**: Real-time progress updates with speed and ETA calculations
- **Zip Compression**: Automatically compress downloaded folders into zip files
- **Cleanup Options**: Option to remove source directory after zip creation
- **Duplicate File Handling**: Smart handling of duplicate file names with automatic numbering
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
# Clone from S3 with beautiful progress bars
./download-bucket clone s3://my-bucket/folder/ ./local-folder

# Clone from DigitalOcean Spaces with verbose output
./download-bucket clone --verbose spaces://my-space/data/ ./local-data

# Clone with specific provider
./download-bucket clone --provider=aws s3://my-bucket/images/ ./images
```

### Advanced Usage

```bash
# High concurrency download
./download-bucket clone --concurrency=10 s3://my-bucket/large-folder/ ./local

# Verbose output
./download-bucket clone --verbose s3://my-bucket/folder/ ./local

# Create zip file after download
./download-bucket clone --zip s3://my-bucket/folder/ ./local

# Create zip file with custom path
./download-bucket clone --zip --zip-path=./my-archive.zip s3://my-bucket/folder/ ./local

# Create zip and remove source directory
./download-bucket clone --zip --remove-source s3://my-bucket/folder/ ./local

# Disable duplicate file handling (overwrites existing files)
./download-bucket clone --handle-duplicates=false s3://my-bucket/folder/ ./local

# Use quiet mode for cleaner output (reduces progress bar updates)
./download-bucket clone --quiet s3://my-bucket/folder/ ./local

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
- `--zip`: Create a zip file after downloading
- `--zip-path`: Path for the zip file (default: `<destination>.zip`)
- `--remove-source`: Remove source directory after creating zip
- `--handle-duplicates`: Handle duplicate files by adding numbers (default: true)
- `--quiet`: Use simple progress output instead of progress bar

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

## User Experience & Performance

### Enhanced UI Features

- **🎨 Beautiful Progress Bars**: Real-time visual progress with colorful output
- **📊 Live Statistics**: Download speed, ETA, file counts, and transfer rates
- **✅ Visual Feedback**: Success/error indicators with emoji and color coding
- **📋 Comprehensive Summary**: Detailed completion reports with compression ratios
- **🚀 Smart Progress**: Automatic calculation of speed, ETA, and completion percentage

### Performance Optimizations

- **⚡ Concurrent Downloads**: Configurable concurrency level for maximum speed
- **💾 Memory Efficient**: Streams files directly to disk without memory overhead
- **📈 Progress Tracking**: Real-time download progress with visual feedback
- **🔄 Resilient Downloads**: Handles interrupted downloads and network issues gracefully

### UI Preview

When you run a download, you'll see:

```
╭─────────────────────────────────────────────────╮
│              Download Starting                  │
╰─────────────────────────────────────────────────╯
📁 Source:      s3://my-bucket/data/
📂 Destination: ./local-data
📊 Total Files: 672
💾 Total Size:  122.5 MB
⏰ Started:     14:23:45

Files: 450/672 | Speed: 2.3 MB/s | ETA: 00:32
[████████████████████████████████          ] 75% 92.1 MB/122.5 MB

✅ Download Completed Successfully!

╭─────────────────────────────────────────────────╮
│                   Summary                       │
╰─────────────────────────────────────────────────╯
📊 Files:       672 successful, 0 failed, 672 total
💾 Data:        122.5 MB downloaded
⚡ Speed:       2.1 MB/s average
⏱️  Duration:    01m23s
📦 Zip:         ./my-archive.zip (115.2 MB)
🗜️  Compression: 6.0% saved
🗑️  Cleanup:     Source directory removed
```

## License

This project is licensed under the MIT License. 