# go-ldap-test

A library for testing code against an LDAP directory.

To the author's knowledge, there does not exist an in-memory LDAP server compatible with OpenLDAP for Golang.

Since that's what I needed, but I didn't have time to write it, I did next best thing and wrote some code to do it via `docker` for me.

In your test code, set up something like this:

	import "github.com/scribd/go-testslapd/pkg/testslapd"
	
    var slapd testslapd.*TestSlapd
    var ldapSetup bool

    func TestMain(m *testing.M) {
        setUp()

        code := m.Run()

        tearDown()

        os.Exit(code)
    }

    // runs before the tests start
    func setUp() {
        // start the test server if it's not already running
        if !ldapSetup {
            port, err := freeport.GetFreePort()
            if err != nil {
                fmt.Printf("Error getting free port: %s\n", err)
                os.Exit(1)

            }

            log.Printf("Starting LDAP Server on port %d", port)
            
            org := "scribd"
            base := "dc=scribd,dc=com"
            domain := "scribd.com"
            adminPassword := "letmein"
            bindDn := fmt.Sprintf("cn=admin,dc=scribd,dc=com")

            slapd = testslapd.NewTestSlapd(port, org, base, domain, adminPassword, "", "")

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
                log.Fatal(err)
            }

            err = slapd.Provision()
            if err != nil {
                fmt.Printf("Failed running provisioner: %s", err)
                log.Fatal(err)
            }

            ldapSetup = true
        }
    }

    // runs at the end to clean up
    func tearDown() {
        err := slapd.StopTestServer()
        if err != nil {
            log.Fatalf(err.Error())
        }
    }

    // run a test that connects to the server
    func TestSomething(t *testing.T)  {
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

        inputs := []struct{
            name string
            dn string
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
            t.Run(tc.name, func(t *testing.T){
                assert.True(t, resp.Entries[i].DN == tc.dn, "Missing expected DN in directory")
            })
        }

        l.Close()
    }
## Caveats

As currently written, only 1 container can run at a time.

If your tests panic, you will likely need to clean out the test container manualy via `docker rm -f ldaptest`.

This package does not install docker for you.  It assumes it's already present, and in the PATH.
