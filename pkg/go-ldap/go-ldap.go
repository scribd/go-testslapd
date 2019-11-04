package go_ldap

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"time"
)

const DEFAULT_CONTAINER_NAME = "ldaptest"
const DEFAULT_CONTAINER_IMAGE = "osixia/openldap"

// TestSlapd a struct representing a locally running docker container containing a running instance of OpenLDAP
type TestSlapd struct {
	Port           int
	Address        string
	Org            string
	Base           string
	Domain         string
	AdminPassword  string
	ContainerName  string
	ContainerImage string
	Verbose        bool
	Provisioner    func() error
}

// NewTestSlapd Create a new TestSlapd object with some defaults.  Default container name is 'ldaptest', default container image is 'osixia/openldap', and default address is '127.0.0.1'.
func NewTestSlapd(port int, org string, base string, domain string, adminPassword string, containerName string, containerImage string) (s *TestSlapd) {
	if containerName == "" {
		containerName = DEFAULT_CONTAINER_NAME
	}

	if containerImage == "" {
		containerImage = DEFAULT_CONTAINER_IMAGE
	}

	address := fmt.Sprintf("127.0.0.1:%d", port)

	s = &TestSlapd{
		Port:           port,
		Address:        address,
		Org:            org,
		Base:           base,
		Domain:         domain,
		AdminPassword:  adminPassword,
		ContainerName:  containerName,
		ContainerImage: containerImage,
		Verbose:        false,
		Provisioner:    nil,
	}

	return s
}

// SetProvisioner  Set's the provisioner function for the test server.
func (s *TestSlapd) SetProvisioner(f func() error) {
	s.Provisioner = f
}

// SetVerbose flips the verbose switch on the test server
func (s *TestSlapd) SetVerbose(b bool) {
	s.Verbose = b
}

// VerboseOutput Convenience function so that I don't have to write 'if s.Verbose {....}' all the time.
func (s *TestSlapd) VerboseOutput(message string, args ...interface{}) {
	if s.Verbose {
		if len(args) == 0 {
			fmt.Printf("%s\n", message)
			return
		}

		msg := fmt.Sprintf(message, args...)
		fmt.Printf("%s\n", msg)
	}
}

// StartTestServer Starts the LDAP directory container
func (s *TestSlapd) StartTestServer() (err error) {
	s.VerboseOutput("Spinning up docker container named %s on port %d", s.ContainerName, s.Port)

	docker, err := exec.LookPath("docker")
	if err != nil {
		log.Fatalf("docker command not found in path")
	}

	ldapOrg := fmt.Sprintf("LDAP_ORGANIZATION=%s", s.Org)
	ldapDomain := fmt.Sprintf("LDAP_DOMAIN=%s", s.Domain)
	ldapAdminPassword := fmt.Sprintf("LDAP_ADMIN_PASSWORD=%s", s.AdminPassword)
	ldapImage := s.ContainerImage
	ldapName := fmt.Sprintf("--name=%s", s.ContainerName)

	portMap := fmt.Sprintf("%d:389", s.Port)

	args := []string{
		"pull",
		ldapImage,
	}

	cmd := exec.Command(docker, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	s.VerboseOutput("Pulling image %s", s.ContainerImage)

	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "failed pulling container")
		return err
	}

	args = []string{
		"run",
		"-e", ldapOrg,
		"-e", ldapDomain,
		"-e", ldapAdminPassword,
		"-p", portMap,
		ldapName,
		"--rm",
		ldapImage,
	}

	s.VerboseOutput("Command: docker %s", args)

	cmd = exec.Command(docker, args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	go cmd.Run()

	s.VerboseOutput("Sleeping for 5 seconds to let the container come up.")

	time.Sleep(5 * time.Second)

	s.VerboseOutput("Container should be up.  Moving on.")

	return err
}

// StopTestServer Stops the LDAP directory container
func (s *TestSlapd) StopTestServer() (err error) {
	s.VerboseOutput("Stopping %s.", s.ContainerName)
	docker, err := exec.LookPath("docker")
	if err != nil {
		return err
	}

	args := []string{
		"stop",
		s.ContainerName,
	}

	cmd := exec.Command(docker, args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()

	return err
}

// Provision Runs the provisioning function.
func (s *TestSlapd) Provision() error {
	return s.Provisioner()
}
