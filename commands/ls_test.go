package commands

import (
	"os"
	"testing"

	"time"

	"errors"

	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/mcndockerclient"
	"github.com/docker/machine/libmachine/state"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/stretchr/testify/assert"
)

func TestParseFiltersErrorsGivenInvalidFilter(t *testing.T) {
	_, err := parseFilters([]string{"foo=bar"})
	assert.EqualError(t, err, "Unsupported filter key 'foo'")
}

func TestParseFiltersSwarm(t *testing.T) {
	actual, _ := parseFilters([]string{"swarm=foo"})
	assert.Equal(t, actual, FilterOptions{SwarmName: []string{"foo"}})
}

func TestParseFiltersDriver(t *testing.T) {
	actual, _ := parseFilters([]string{"driver=bar"})
	assert.Equal(t, actual, FilterOptions{DriverName: []string{"bar"}})
}

func TestParseFiltersState(t *testing.T) {
	actual, _ := parseFilters([]string{"state=Running"})
	assert.Equal(t, actual, FilterOptions{State: []string{"Running"}})
}

func TestParseFiltersName(t *testing.T) {
	actual, _ := parseFilters([]string{"name=dev"})
	assert.Equal(t, actual, FilterOptions{Name: []string{"dev"}})
}

func TestParseFiltersAll(t *testing.T) {
	actual, _ := parseFilters([]string{"swarm=foo", "driver=bar", "state=Stopped", "name=dev"})
	assert.Equal(t, actual, FilterOptions{SwarmName: []string{"foo"}, DriverName: []string{"bar"}, State: []string{"Stopped"}, Name: []string{"dev"}})
}

func TestParseFiltersAllCase(t *testing.T) {
	actual, err := parseFilters([]string{"sWarM=foo", "DrIver=bar", "StaTe=Stopped", "NAMe=dev"})
	assert.Equal(t, actual, FilterOptions{SwarmName: []string{"foo"}, DriverName: []string{"bar"}, State: []string{"Stopped"}, Name: []string{"dev"}})
	assert.Nil(t, err, "err should be nil")
}

func TestParseFiltersDuplicates(t *testing.T) {
	actual, _ := parseFilters([]string{"swarm=foo", "driver=bar", "name=mark", "swarm=baz", "driver=qux", "state=Running", "state=Starting", "name=time"})
	assert.Equal(t, actual, FilterOptions{SwarmName: []string{"foo", "baz"}, DriverName: []string{"bar", "qux"}, State: []string{"Running", "Starting"}, Name: []string{"mark", "time"}})
}

func TestParseFiltersValueWithEqual(t *testing.T) {
	actual, _ := parseFilters([]string{"driver=bar=baz"})
	assert.Equal(t, actual, FilterOptions{DriverName: []string{"bar=baz"}})
}

func TestFilterHostsReturnsFiltersValuesCaseInsensitive(t *testing.T) {
	opts := FilterOptions{
		SwarmName:  []string{"fOo"},
		DriverName: []string{"ViRtUaLboX"},
		State:      []string{"StOPpeD"},
	}
	hosts := []*host.Host{}
	actual := filterHosts(hosts, opts)
	assert.EqualValues(t, actual, hosts)
}
func TestFilterHostsReturnsSameGivenNoFilters(t *testing.T) {
	opts := FilterOptions{}
	hosts := []*host.Host{
		{
			Name:        "testhost",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
		},
	}
	actual := filterHosts(hosts, opts)
	assert.EqualValues(t, actual, hosts)
}

func TestFilterHostsReturnsEmptyGivenEmptyHosts(t *testing.T) {
	opts := FilterOptions{
		SwarmName: []string{"foo"},
	}
	hosts := []*host.Host{}
	assert.Empty(t, filterHosts(hosts, opts))
}

func TestFilterHostsReturnsEmptyGivenNonMatchingFilters(t *testing.T) {
	opts := FilterOptions{
		SwarmName: []string{"foo"},
	}
	hosts := []*host.Host{
		{
			Name:        "testhost",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
		},
	}
	assert.Empty(t, filterHosts(hosts, opts))
}

func TestFilterHostsBySwarmName(t *testing.T) {
	opts := FilterOptions{
		SwarmName: []string{"master"},
	}
	master :=
		&host.Host{
			Name: "master",
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{Master: true, Discovery: "foo"},
			},
		}
	node1 :=
		&host.Host{
			Name: "node1",
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{Master: false, Discovery: "foo"},
			},
		}
	othermaster :=
		&host.Host{
			Name: "othermaster",
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{Master: true, Discovery: "bar"},
			},
		}
	hosts := []*host.Host{master, node1, othermaster}
	expected := []*host.Host{master, node1}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestFilterHostsByDriverName(t *testing.T) {
	opts := FilterOptions{
		DriverName: []string{"fakedriver"},
	}
	node1 :=
		&host.Host{
			Name:        "node1",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
		}
	node2 :=
		&host.Host{
			Name:        "node2",
			DriverName:  "virtualbox",
			HostOptions: &host.Options{},
		}
	node3 :=
		&host.Host{
			Name:        "node3",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
		}
	hosts := []*host.Host{node1, node2, node3}
	expected := []*host.Host{node1, node3}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestFilterHostsByState(t *testing.T) {
	opts := FilterOptions{
		State: []string{"Paused", "Saved", "Stopped"},
	}
	node1 :=
		&host.Host{
			Name:        "node1",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused},
		}
	node2 :=
		&host.Host{
			Name:        "node2",
			DriverName:  "virtualbox",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Stopped},
		}
	node3 :=
		&host.Host{
			Name:        "node3",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Running},
		}
	hosts := []*host.Host{node1, node2, node3}
	expected := []*host.Host{node1, node2}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestFilterHostsByName(t *testing.T) {
	opts := FilterOptions{
		Name: []string{"fire", "ice", "earth", "a.?r"},
	}
	node1 :=
		&host.Host{
			Name:        "fire",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused, MockName: "fire"},
		}
	node2 :=
		&host.Host{
			Name:        "ice",
			DriverName:  "adriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused, MockName: "ice"},
		}
	node3 :=
		&host.Host{
			Name:        "air",
			DriverName:  "nodriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused, MockName: "air"},
		}
	node4 :=
		&host.Host{
			Name:        "water",
			DriverName:  "falsedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused, MockName: "water"},
		}
	hosts := []*host.Host{node1, node2, node3, node4}
	expected := []*host.Host{node1, node2, node3}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestFilterHostsMultiFlags(t *testing.T) {
	opts := FilterOptions{
		SwarmName:  []string{},
		DriverName: []string{"fakedriver", "virtualbox"},
	}
	node1 :=
		&host.Host{
			Name:        "node1",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
		}
	node2 :=
		&host.Host{
			Name:        "node2",
			DriverName:  "virtualbox",
			HostOptions: &host.Options{},
		}
	node3 :=
		&host.Host{
			Name:        "node3",
			DriverName:  "softlayer",
			HostOptions: &host.Options{},
		}
	hosts := []*host.Host{node1, node2, node3}
	expected := []*host.Host{node1, node2}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestFilterHostsDifferentFlagsProduceAND(t *testing.T) {
	opts := FilterOptions{
		DriverName: []string{"virtualbox"},
		State:      []string{"Running"},
	}
	node1 :=
		&host.Host{
			Name:        "node1",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Paused},
		}
	node2 :=
		&host.Host{
			Name:        "node2",
			DriverName:  "virtualbox",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Stopped},
		}
	node3 :=
		&host.Host{
			Name:        "node3",
			DriverName:  "fakedriver",
			HostOptions: &host.Options{},
			Driver:      &fakedriver.Driver{MockState: state.Running},
		}
	hosts := []*host.Host{node1, node2, node3}
	expected := []*host.Host{}

	assert.EqualValues(t, filterHosts(hosts, opts), expected)
}

func TestGetHostListItems(t *testing.T) {
	mcndockerclient.CurrentDockerVersioner = &mcndockerclient.FakeDockerVersioner{Version: "1.9"}
	defer mcndockerclient.CleanupDockerVersioner()

	// TODO: Ideally this would mockable via interface instead.
	os.Setenv("DOCKER_HOST", "tcp://active.host.com:2376")

	hosts := []*host.Host{
		{
			Name: "foo",
			Driver: &fakedriver.Driver{
				MockState: state.Running,
				MockIP:    "active.host.com",
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{},
			},
		},
		{
			Name: "bar100",
			Driver: &fakedriver.Driver{
				MockState: state.Stopped,
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{},
			},
		},
		{
			Name: "bar10",
			Driver: &fakedriver.Driver{
				MockState: state.Error,
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{},
			},
		},
	}

	expected := []struct {
		name    string
		state   state.State
		active  bool
		version string
		error   string
	}{
		{"bar10", state.Error, false, "Unknown", "Unable to get ip"},
		{"bar100", state.Stopped, false, "Unknown", ""},
		{"foo", state.Running, true, "v1.9", ""},
	}

	items := getHostListItems(hosts, map[string]error{})

	for i := range expected {
		assert.Equal(t, expected[i].name, items[i].Name)
		assert.Equal(t, expected[i].state, items[i].State)
		assert.Equal(t, expected[i].active, items[i].Active)
		assert.Equal(t, expected[i].version, items[i].DockerVersion)
		assert.Equal(t, expected[i].error, items[i].Error)
	}

	os.Unsetenv("DOCKER_HOST")
}

// issue #1908
func TestGetHostListItemsEnvDockerHostUnset(t *testing.T) {
	mcndockerclient.CurrentDockerVersioner = &mcndockerclient.FakeDockerVersioner{Version: "1.9"}
	defer mcndockerclient.CleanupDockerVersioner()

	orgDockerHost := os.Getenv("DOCKER_HOST")
	defer func() {
		// revert DOCKER_HOST
		os.Setenv("DOCKER_HOST", orgDockerHost)
	}()

	// unset DOCKER_HOST
	os.Unsetenv("DOCKER_HOST")

	hosts := []*host.Host{
		{
			Name: "foo",
			Driver: &fakedriver.Driver{
				MockState: state.Running,
				MockIP:    "120.0.0.1",
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{
					Master:    false,
					Address:   "",
					Discovery: "",
				},
			},
		},
		{
			Name: "bar",
			Driver: &fakedriver.Driver{
				MockState: state.Stopped,
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{
					Master:    false,
					Address:   "",
					Discovery: "",
				},
			},
		},
		{
			Name: "baz",
			Driver: &fakedriver.Driver{
				MockState: state.Saved,
			},
			HostOptions: &host.Options{
				SwarmOptions: &swarm.Options{
					Master:    false,
					Address:   "",
					Discovery: "",
				},
			},
		},
	}

	expected := map[string]struct {
		state   state.State
		active  bool
		version string
	}{
		"foo": {state.Running, false, "v1.9"},
		"bar": {state.Stopped, false, "Unknown"},
		"baz": {state.Saved, false, "Unknown"},
	}

	items := getHostListItems(hosts, map[string]error{})

	for _, item := range items {
		expected := expected[item.Name]

		assert.Equal(t, expected.state, item.State)
		assert.Equal(t, expected.active, item.Active)
		assert.Equal(t, expected.version, item.DockerVersion)
	}
}

func TestIsActive(t *testing.T) {
	cases := []struct {
		dockerHost string
		state      state.State
		expected   bool
	}{
		{"", state.Running, false},
		{"tcp://5.6.7.8:2376", state.Running, false},
		{"tcp://1.2.3.4:2376", state.Stopped, false},
		{"tcp://1.2.3.4:2376", state.Running, true},
		{"tcp://1.2.3.4:3376", state.Running, true},
	}

	for _, c := range cases {
		os.Unsetenv("DOCKER_HOST")
		if c.dockerHost != "" {
			os.Setenv("DOCKER_HOST", c.dockerHost)
		}

		actual := isActive(c.state, "tcp://1.2.3.4:2376")

		assert.Equal(t, c.expected, actual, "IsActive(%s, \"%s\") should return %v, but didn't", c.state, c.dockerHost, c.expected)
	}
}

func TestGetHostStateTimeout(t *testing.T) {
	originalTimeout := stateTimeoutDuration

	hosts := []*host.Host{
		{
			Name: "foo",
			Driver: &fakedriver.Driver{
				MockState: state.Timeout,
			},
		},
	}

	stateTimeoutDuration = 1 * time.Millisecond
	hostItems := getHostListItems(hosts, map[string]error{})
	hostItem := hostItems[0]

	assert.Equal(t, "foo", hostItem.Name)
	assert.Equal(t, state.Timeout, hostItem.State)
	assert.Equal(t, "Driver", hostItem.DriverName)

	stateTimeoutDuration = originalTimeout
}

func TestGetHostStateError(t *testing.T) {
	hosts := []*host.Host{
		{
			Name: "foo",
			Driver: &fakedriver.Driver{
				MockState: state.Error,
			},
		},
	}

	hostItems := getHostListItems(hosts, map[string]error{})
	hostItem := hostItems[0]

	assert.Equal(t, "foo", hostItem.Name)
	assert.Equal(t, state.Error, hostItem.State)
	assert.Equal(t, "Driver", hostItem.DriverName)
	assert.Empty(t, hostItem.URL)
	assert.Equal(t, "Unable to get ip", hostItem.Error)
	assert.Nil(t, hostItem.SwarmOptions)
}

func TestGetSomeHostInEror(t *testing.T) {
	mcndockerclient.CurrentDockerVersioner = &mcndockerclient.FakeDockerVersioner{Version: "1.9"}
	defer mcndockerclient.CleanupDockerVersioner()

	hosts := []*host.Host{
		{
			Name: "foo",
			Driver: &fakedriver.Driver{
				MockState: state.Running,
			},
		},
	}
	hostsInError := map[string]error{
		"bar": errors.New("invalid memory address or nil pointer dereference"),
	}

	hostItems := getHostListItems(hosts, hostsInError)
	assert.Equal(t, 2, len(hostItems))

	hostItem := hostItems[0]

	assert.Equal(t, "bar", hostItem.Name)
	assert.Equal(t, state.Error, hostItem.State)
	assert.Equal(t, "not found", hostItem.DriverName)
	assert.Empty(t, hostItem.URL)
	assert.Equal(t, "invalid memory address or nil pointer dereference", hostItem.Error)
	assert.Nil(t, hostItem.SwarmOptions)

	hostItem = hostItems[1]

	assert.Equal(t, "foo", hostItem.Name)
	assert.Equal(t, state.Running, hostItem.State)
}
