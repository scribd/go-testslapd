package testslapd

import (
	"fmt"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ldap.v2"
	"log"
	"testing"
)

func TestSlapdTestServer(t *testing.T) {
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatalf("Failed to get a free port on which to run the test vault server: %s", err)
	}

	org := "scribd"
	base := "dc=scribd,dc=com"
	domain := "scribd.com"
	adminPassword := "letmein"
	bindDn := fmt.Sprintf("cn=admin,dc=scribd,dc=com")

	slapd := NewTestSlapd(port, org, base, domain, adminPassword, "", "")

	slapd.SetVerbose(true)

	slapd.SetProvisioner(func() error {
		l, err := ldap.Dial("tcp", slapd.Address)
		if err != nil {
			fmt.Printf("Failed to dial LDAP at %s: %s", slapd.Address, err)
			t.Fail()
		}

		err = l.Bind(bindDn, adminPassword)
		if err != nil {
			log.Printf("--- failed to bind to ldap: %s ---", err)
			return err
		}

		fmt.Printf("--- Running provision function ---\n")

		log.Printf("--- adding group ou ---")
		r := ldap.NewAddRequest("ou=group,dc=scribd,dc=com")
		r.Attribute("ou", []string{"group"})
		r.Attribute("objectClass", []string{"top", "organizationalUnit"})

		err = l.Add(r)
		if err != nil {
			log.Printf("failed to add group ou to directory: %s", err)
			t.Fail()
		}

		r = ldap.NewAddRequest("ou=users,dc=scribd,dc=com")
		r.Attribute("ou", []string{"users"})
		r.Attribute("objectClass", []string{"top", "organizationalUnit"})

		log.Printf("--- adding users ou ---")

		err = l.Add(r)
		if err != nil {
			log.Printf("failed to add users ou to directory: %s", err)
			t.Fail()
		}

		l.Close()

		fmt.Printf("--- End provision function ---\n")

		return err
	})

	err = slapd.StartTestServer()
	if err != nil {
		fmt.Printf("Failed starting slapd container %q: %s", slapd.ContainerName, err)
		t.Fail()
	}

	err = slapd.Provision()
	if err != nil {
		fmt.Printf("Failed running provisioner: %s", err)
		t.Fail()
	}

	l, err := ldap.Dial("tcp", slapd.Address)
	if err != nil {
		fmt.Printf("Failed to dial LDAP at %s: %s", slapd.Address, err)
		t.Fail()
	}

	err = l.Bind(bindDn, adminPassword)
	if err != nil {
		log.Printf("--- failed to bind to ldap: %s ---", err)
		t.Fail()
	}

	req := ldap.NewSearchRequest(
		base,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=top)",
		[]string{},
		nil,
	)

	resp, err := l.Search(req)
	if err != nil {
		fmt.Printf("Failed searching directory: %s", err)
		t.Fail()
	}
	assert.True(t, len(resp.Entries) == 4, "Expected entries in directory")

	inputs := []struct {
		name string
		dn   string
	}{
		{
			"org",
			"dc=scribd,dc=com",
		},
		{
			"admin",
			"cn=admin,dc=scribd,dc=com",
		},
		{
			"group",
			"ou=group,dc=scribd,dc=com",
		},
		{
			"users",
			"ou=users,dc=scribd,dc=com",
		},
	}

	for i, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, resp.Entries[i].DN == tc.dn, "Missing expected DN in directory")
		})
	}

	l.Close()

	err = slapd.StopTestServer()
	if err != nil {
		fmt.Printf("Failed stopping slapd container %s: %s", slapd.ContainerName, err)
	}
}
