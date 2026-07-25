// Package valkey only exists to register the valkey extension
package valkey

import (
	"github.com/moko-poi/xk6-valkey/valkey"
	"go.k6.io/k6/v2/js/modules"
)

// Register the extension on module initialization, available to
// import from JS as "k6/x/valkey".
func init() {
	modules.Register("k6/x/valkey", valkey.New())
}
