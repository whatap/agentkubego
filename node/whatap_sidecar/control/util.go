package control

import (
	"github.com/whatap/golib/lang/pack"
	"whatap.io/k8s/sidecar/session"
)

func sendHide(p pack.Pack) bool {
	// b := pack.ToBytesPack(p)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Hide(b)
	// return sendSecure(net.NET_SECURE_HIDE, b)
	// return sendSecure(0, b)
	return session.SendHide(p)
}

func sendEncrypted(p pack.Pack) bool {
	// b := pack.ToBytesPack(p)
	// return sendSecure(0, b)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Encrypt(b)
	// return sendSecure(net.NET_SECURE_CYPHER, b)
	return session.SendEncrypted(p)
}
