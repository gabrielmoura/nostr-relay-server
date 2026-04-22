package groups

import (
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

func configRoleMember() config.NIP29RoleConfig {
	return config.NIP29RoleConfig{
		Name:        "member",
		Description: "Default group member without elevated permissions",
		Permissions: []string{},
	}
}

func errEnsureRole(name string, err error) error {
	return fmt.Errorf("ensuring nip29 role %s: %w", name, err)
}

func errMissingCreatorRole(name string) error {
	return errors.New("nip29 group_creator_role \"" + name + "\" is not present in default_roles")
}
