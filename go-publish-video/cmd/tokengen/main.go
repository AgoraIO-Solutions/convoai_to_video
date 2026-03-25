package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	rtctokenbuilder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
)

func main() {
	var (
		appID          string
		appCert        string
		channelName    string
		uid            string
		role           string
		enableStringUID bool
	)

	flag.StringVar(&appID, "appID", "", "Agora App ID")
	flag.StringVar(&appCert, "appCert", "", "Agora App Certificate")
	flag.StringVar(&channelName, "channelName", "", "Channel name")
	flag.StringVar(&uid, "uid", "", "User ID (numeric for int UID mode, any string for string UID mode)")
	flag.StringVar(&role, "role", "publisher", "Role: publisher or subscriber")
	flag.BoolVar(&enableStringUID, "enableStringUID", true, "Use string UID token (BuildTokenWithUserAccount) vs int UID token (BuildTokenWithUid)")
	flag.Parse()

	if appID == "" || appCert == "" || channelName == "" || uid == "" {
		fmt.Fprintln(os.Stderr, "Error: -appID, -appCert, -channelName, and -uid are required")
		flag.Usage()
		os.Exit(1)
	}

	var tokenRole rtctokenbuilder.Role
	switch role {
	case "publisher":
		tokenRole = rtctokenbuilder.RolePublisher
	case "subscriber":
		tokenRole = rtctokenbuilder.RoleSubscriber
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid role %q, must be 'publisher' or 'subscriber'\n", role)
		os.Exit(1)
	}

	var token string
	var err error

	if enableStringUID {
		token, err = rtctokenbuilder.BuildTokenWithUserAccount(
			appID, appCert, channelName, uid,
			tokenRole, 3600, 3600,
		)
	} else {
		uidInt, parseErr := strconv.ParseUint(uid, 10, 32)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: -uid must be a numeric value when -enableStringUID=false, got %q\n", uid)
			os.Exit(1)
		}
		token, err = rtctokenbuilder.BuildTokenWithUid(
			appID, appCert, channelName, uint32(uidInt),
			tokenRole, 3600, 3600,
		)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(token)
}
