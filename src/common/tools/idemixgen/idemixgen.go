/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

// idemixgen is a command line tool that generates the CA's keys and
// generates MSP configs for siging and for verification
// This tool can be used to setup the peers and CA to support
// the Identity Mixer MSP

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/foglink/fnkcore/src/common/tools/idemixgen/idemixca"
	"github.com/foglink/fnkcore/src/idemix"
	"github.com/foglink/fnkcore/src/msp"
	"github.com/foglink/fnkcore/src/orderer/common/metadata"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	IdemixDirIssuer             = "ca"
	IdemixConfigIssuerSecretKey = "IssuerSecretKey"
)

// command line flags
var (
	app = kingpin.New("idemixgen", "Utility for generating key material to be used with the Identity Mixer MSP in foglink fnkcore")

	genIssuerKey    = app.Command("ca-keygen", "Generate CA key material")
	genSignerConfig = app.Command("signerconfig", "Generate a default signer for this Idemix MSP")
	genCredOU       = genSignerConfig.Flag("org-unit", "The Organizational Unit of the default signer").Short('u').String()
	genCredIsAdmin  = genSignerConfig.Flag("admin", "Make the default signer admin").Short('a').Bool()

	version = app.Command("version", "Show version information")
)

func main() {
	app.HelpFlag.Short('h')

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case genIssuerKey.FullCommand():
		isk, ipk, err := idemixca.GenerateIssuerKey()
		handleError(err)

		// Prevent overwriting the existing key
		path := filepath.Join(IdemixDirIssuer)
		checkDirectoryNotExists(path, fmt.Sprintf("Directory %s already exists", path))

		path = msp.IdemixConfigDirMsp
		checkDirectoryNotExists(path, fmt.Sprintf("Directory %s already exists", path))

		// write private and public keys to the file
		handleError(os.Mkdir(IdemixDirIssuer, 0770))
		handleError(os.Mkdir(msp.IdemixConfigDirMsp, 0770))
		writeFile(filepath.Join(IdemixDirIssuer, IdemixConfigIssuerSecretKey), isk)
		writeFile(filepath.Join(IdemixDirIssuer, msp.IdemixConfigFileIssuerPublicKey), ipk)
		writeFile(filepath.Join(msp.IdemixConfigDirMsp, msp.IdemixConfigFileIssuerPublicKey), ipk)

	case genSignerConfig.FullCommand():
		config, err := idemixca.GenerateSignerConfig(*genCredIsAdmin, *genCredOU, readIssuerKey())
		handleError(err)

		path := msp.IdemixConfigDirUser
		checkDirectoryNotExists(path, fmt.Sprintf("This MSP config already contains a directory \"%s\"", path))

		// Write config to file
		handleError(os.Mkdir(msp.IdemixConfigDirUser, 0770))
		writeFile(filepath.Join(msp.IdemixConfigDirUser, msp.IdemixConfigFileSigner), config)

	case version.FullCommand():
		printVersion()
	}
}

func printVersion() {
	fmt.Println(metadata.GetVersionInfo())
}

// writeFile writes bytes to a file and panics in case of an error
func writeFile(path string, contents []byte) {
	handleError(ioutil.WriteFile(path, contents, 0640))
}

// readIssuerKey reads the issuer key from the current directory
func readIssuerKey() *idemix.IssuerKey {
	path := filepath.Join(IdemixDirIssuer, IdemixConfigIssuerSecretKey)
	isk, err := ioutil.ReadFile(path)
	if err != nil {
		handleError(errors.Wrapf(err, "failed to open issuer secret key file: %s", path))
	}
	path = filepath.Join(IdemixDirIssuer, msp.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		handleError(errors.Wrapf(err, "failed to open issuer public key file: %s", path))
	}
	ipk := &idemix.IssuerPublicKey{}
	handleError(proto.Unmarshal(ipkBytes, ipk))
	key := &idemix.IssuerKey{isk, ipk}

	return key
}

// checkDirectoryNotExists checks whether a directory with the given path already exists and exits if this is the case
func checkDirectoryNotExists(path string, errorMessage string) {
	_, err := os.Stat(path)
	if err == nil {
		handleError(errors.New(errorMessage))
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}