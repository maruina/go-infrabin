package infrabin

type Config struct {
	EnableProxyEndpoint bool
	AWSMetadataEndpoint string
}

func DefaultConfig() *Config {
	return &Config{
		EnableProxyEndpoint: false,
		AWSMetadataEndpoint: "http://169.254.169.254/latest/meta-data/",
	}
}
