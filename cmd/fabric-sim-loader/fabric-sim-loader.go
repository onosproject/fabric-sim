// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"github.com/onosproject/fabric-sim/pkg/loader"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
)

const (
	addressFlag     = "service-address"
	tlsCertPathFlag = "tls-cert-path"
	tlsKeyPathFlag  = "tls-key-path"
	noTLSFlag       = "no-tls"
	topologyFlag    = "topology"
)

// The main entry point
func main() {
	if err := getRootCommand().Execute(); err != nil {
		println(err)
		os.Exit(1)
	}
}

func getRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fabric-sim-loader",
		Short: "loader",
		RunE:  runRootCommand,
	}
	cmd.Flags().String(addressFlag, "fabric-sim:5150", "service address")
	cmd.Flags().String(tlsKeyPathFlag, "", "path to client private key")
	cmd.Flags().String(tlsCertPathFlag, "", "path to client certificate")
	cmd.Flags().String(topologyFlag, "-", "topology YAML file")
	return cmd
}

func getAddress(cmd *cobra.Command) string {
	address, _ := cmd.Flags().GetString(addressFlag)
	return address
}

func getCertPath(cmd *cobra.Command) string {
	certPath, _ := cmd.Flags().GetString(tlsCertPathFlag)
	return certPath
}

func getKeyPath(cmd *cobra.Command) string {
	keyPath, _ := cmd.Flags().GetString(tlsKeyPathFlag)
	return keyPath
}

func noTLS(cmd *cobra.Command) bool {
	tls, _ := cmd.Flags().GetBool(noTLSFlag)
	return tls
}

// getConnection returns a gRPC client connection to the onos service
func getConnection(cmd *cobra.Command) (*grpc.ClientConn, error) {
	address := getAddress(cmd)
	certPath := getCertPath(cmd)
	keyPath := getKeyPath(cmd)
	var opts []grpc.DialOption

	if noTLS(cmd) {
		opts = []grpc.DialOption{
			grpc.WithInsecure(),
		}
	} else {
		if certPath != "" && keyPath != "" {
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				return nil, err
			}
			opts = []grpc.DialOption{
				grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
				})),
			}
		} else {
			// Load default Certificates
			cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
			if err != nil {
				return nil, err
			}
			opts = []grpc.DialOption{
				grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
				})),
			}
		}
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func runRootCommand(cmd *cobra.Command, args []string) error {
	conn, err := getConnection(cmd)
	defer func() { _ = conn.Close() }()

	if err != nil {
		return err
	}

	topologyPath, _ := cmd.Flags().GetString(topologyFlag)
	if topologyPath == "-" {

	}
	return loader.LoadTopology(conn, topologyPath)
}
