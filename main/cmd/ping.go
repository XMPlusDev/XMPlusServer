package cmd

import (
	gotls "crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	
	"github.com/spf13/cobra"
	. "github.com/xtls/xray-core/transport/internet/tls"
)

var (
	tlsPingIP  string
	tlsPingCmd = &cobra.Command{
		Use:   "ping <domain>",
		Short: "Ping the domain with TLS handshake",
		Long: `Ping the domain with TLS handshake.

The command performs TLS handshakes both with and without SNI to test
the TLS configuration of the target domain.

Examples:
  ping example.com
  ping example.com:8443
  ping -ip 1.2.3.4 example.com`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			executeTLSPing(args[0])
		},
	}
)

func init() {
	tlsPingCmd.Flags().StringVar(&tlsPingIP, "ip", "", "The IP address of the domain")
	// Assuming there's a parent 'tls' command
	// tlsCmd.AddCommand(tlsPingCmd)
	// If this is a root-level command, use:
	rootCmd.AddCommand(tlsPingCmd)
}

func executeTLSPing(domainWithPort string) error {
	fmt.Println("TLS ping: ", domainWithPort)
	
	TargetPort := 443
	domain, port, err := net.SplitHostPort(domainWithPort)
	if err != nil {
		domain = domainWithPort
	} else {
		TargetPort, _ = strconv.Atoi(port)
	}

	var ip net.IP
	if len(tlsPingIP) > 0 {
		v := net.ParseIP(tlsPingIP)
		if v == nil {
			fmt.Printf("Error: invalid IP: %s\n", tlsPingIP)
			return nil
		}
		ip = v
	} else {
		v, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			fmt.Printf("Error: Failed to resolve IP: %s\n", err)
			return nil
		}
		ip = v.IP
	}
	fmt.Println("Using IP: ", ip.String()+":"+strconv.Itoa(TargetPort))

	fmt.Println("-------------------")
	fmt.Println("Pinging without SNI")
	{
		tcpConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ip, Port: TargetPort})
		if err != nil {
			fmt.Printf("Failed to dial tcp: %s\n", err)
		} else {
			tlsConn := gotls.Client(tcpConn, &gotls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h2", "http/1.1"},
				MaxVersion:         gotls.VersionTLS13,
				MinVersion:         gotls.VersionTLS12,
				// Do not release tool before v5's refactor
				// VerifyPeerCertificate: showCert(),
			})
			err = tlsConn.Handshake()
			if err != nil {
				fmt.Println("Handshake failure: ", err)
			} else {
				fmt.Println("Handshake succeeded")
				printTLSConnDetail(tlsConn)
				printCertificates(tlsConn.ConnectionState().PeerCertificates)
			}
			tlsConn.Close()
		}
	}

	fmt.Println("-------------------")
	fmt.Println("Pinging with SNI")
	{
		tcpConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ip, Port: TargetPort})
		if err != nil {
			fmt.Printf("Failed to dial tcp: %s\n", err)
		} else {
			tlsConn := gotls.Client(tcpConn, &gotls.Config{
				ServerName: domain,
				NextProtos: []string{"h2", "http/1.1"},
				MaxVersion: gotls.VersionTLS13,
				MinVersion: gotls.VersionTLS12,
				// Do not release tool before v5's refactor
				// VerifyPeerCertificate: showCert(),
			})
			err = tlsConn.Handshake()
			if err != nil {
				fmt.Println("Handshake failure: ", err)
			} else {
				fmt.Println("Handshake succeeded")
				printTLSConnDetail(tlsConn)
				printCertificates(tlsConn.ConnectionState().PeerCertificates)
			}
			tlsConn.Close()
		}
	}

	fmt.Println("-------------------")
	fmt.Println("TLS ping finished")
	
	return nil
}

func printCertificates(certs []*x509.Certificate) {
	var leaf *x509.Certificate
	var length int
	for _, cert := range certs {
		length += len(cert.Raw)
		if len(cert.DNSNames) != 0 {
			leaf = cert
		}
	}
	fmt.Println("Certificate chain's total length: ", length, "(certs count: "+strconv.Itoa(len(certs))+")")
	if leaf != nil {
		fmt.Println("Cert's signature algorithm: ", leaf.SignatureAlgorithm.String())
		fmt.Println("Cert's publicKey algorithm: ", leaf.PublicKeyAlgorithm.String())
		fmt.Println("Cert's allowed domains: ", leaf.DNSNames)
	}
}

func printTLSConnDetail(tlsConn *gotls.Conn) {
	connectionState := tlsConn.ConnectionState()
	var tlsVersion string
	if connectionState.Version == gotls.VersionTLS13 {
		tlsVersion = "TLS 1.3"
	} else if connectionState.Version == gotls.VersionTLS12 {
		tlsVersion = "TLS 1.2"
	}
	fmt.Println("TLS Version: ", tlsVersion)
	curveID := connectionState.CurveID
	if curveID != 0 {
		PostQuantum := (curveID == gotls.X25519MLKEM768)
		fmt.Println("TLS Post-Quantum key exchange: ", PostQuantum, "("+curveID.String()+")")
	} else {
		fmt.Println("TLS Post-Quantum key exchange:  false (RSA Exchange)")
	}
}

func showCert() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		var hash []byte
		for _, asn1Data := range rawCerts {
			cert, _ := x509.ParseCertificate(asn1Data)
			if cert.IsCA {
				hash = GenerateCertHash(cert)
			}
		}
		fmt.Println("Certificate Leaf Hash: ", hex.EncodeToString(hash))
		return nil
	}
}