package utils

import (
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	gogo_types "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

func ResolveUpstreamSslConfig(s *v1.UpstreamSslConfig, secrets v1.SecretList) (*envoyauth.UpstreamTlsContext, error) {
	if common, err := ResolveCommonSslConfig(s, secrets); err == nil {
		return &envoyauth.UpstreamTlsContext{
			CommonTlsContext: common,
			Sni:              s.Sni,
		}, nil
	} else {
		return nil, err
	}
}
func ResolveDownstreamSslConfig(s *v1.SslConfig, secrets v1.SecretList) (*envoyauth.DownstreamTlsContext, error) {
	if common, err := ResolveCommonSslConfig(s, secrets); err == nil {
		var requireClientCert *gogo_types.BoolValue
		if common.ValidationContextType != nil {
			requireClientCert = &gogo_types.BoolValue{Value: true}
		}
		return &envoyauth.DownstreamTlsContext{
			CommonTlsContext:         common,
			RequireClientCertificate: requireClientCert,
		}, nil
	} else {
		return nil, err
	}
}

type CertSource interface {
	GetSecretRef() *core.ResourceRef
	GetSslFiles() *v1.SSLFiles
}

func ResolveCommonSslConfig(s CertSource, secrets v1.SecretList) (*envoyauth.CommonTlsContext, error) {
	var (
		certChain, privateKey, rootCa string
		// if using a Secret ref, we will inline the certs in the tls config
		inlineDataSource bool
	)

	if sslSecrets := s.GetSecretRef(); sslSecrets != nil {
		var err error
		inlineDataSource = true
		ref := sslSecrets
		certChain, privateKey, rootCa, err = GetSslSecrets(*ref, secrets)
		if err != nil {
			return nil, err
		}
	} else if sslSecrets := s.GetSslFiles(); sslSecrets != nil {
		certChain, privateKey, rootCa = sslSecrets.TlsCert, sslSecrets.TlsKey, sslSecrets.RootCa
	} else {
		return nil, errors.New("no certificate information found")
	}

	var certChainData, privateKeyData, rootCaData *envoycore.DataSource
	if !inlineDataSource {
		if certChain != "" {
			certChainData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_Filename{
					Filename: certChain,
				},
			}
		}
		if privateKey != "" {
			privateKeyData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_Filename{
					Filename: privateKey,
				},
			}
		}
		if rootCa != "" {
			rootCaData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_Filename{
					Filename: rootCa,
				},
			}
		}
	} else {
		if certChain != "" {
			certChainData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_InlineString{
					InlineString: certChain,
				},
			}
		}
		if privateKey != "" {
			privateKeyData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_InlineString{
					InlineString: privateKey,
				},
			}
		}
		if rootCa != "" {
			rootCaData = &envoycore.DataSource{
				Specifier: &envoycore.DataSource_InlineString{
					InlineString: rootCa,
				},
			}
		}
	}

	tlsContext := &envoyauth.CommonTlsContext{
		// default params
		TlsParams:     &envoyauth.TlsParameters{},
		AlpnProtocols: []string{"h2", "http/1.1"},
	}

	if certChainData != nil && privateKeyData != nil {
		tlsContext.TlsCertificates = []*envoyauth.TlsCertificate{
			{
				CertificateChain: certChainData,
				PrivateKey:       privateKeyData,
			},
		}
	} else if certChainData != nil || privateKeyData != nil {
		return nil, errors.New("both or none of cert chain and private key must be provided")
	}

	if rootCaData != nil {
		tlsContext.ValidationContextType = &envoyauth.CommonTlsContext_ValidationContext{
			ValidationContext: &envoyauth.CertificateValidationContext{
				TrustedCa: rootCaData,
			},
		}
	}

	return tlsContext, nil
}

func GetSslSecrets(ref core.ResourceRef, secrets v1.SecretList) (string, string, string, error) {
	secret, err := secrets.Find(ref.Strings())
	if err != nil {
		return "", "", "", errors.Wrapf(err, "SSL secret not found")
	}

	sslSecret, ok := secret.Kind.(*v1.Secret_Tls)
	if !ok {
		return "", "", "", errors.Errorf("%v is not a TLS secret", secret.GetMetadata().Ref())
	}

	certChain := sslSecret.Tls.CertChain
	privateKey := sslSecret.Tls.PrivateKey
	rootCa := sslSecret.Tls.RootCa
	return certChain, privateKey, rootCa, nil
}
