package condition

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/brexhq/substation/config"
	"github.com/brexhq/substation/internal/errors"
)

// forEach evaluates conditions by iterating and applying an inspector to each element in a JSON array.
//
// This inspector supports the object handling pattern.
type inspForEach struct {
	condition
	Options inspForEachOptions `json:"options"`

	inspector Inspector
}

type inspForEachOptions struct {
	// Type determines the method of combining results from the inspector.
	//
	// Must be one of:
	//
	// - none: none of the elements match the condition
	//
	// - any: at least one of the elements match the condition
	//
	// - all: all of the elements match the condition
	Type string `json:"type"`
	// Inspector is the condition applied to each element.
	Inspector config.Config `json:"inspector"`
}

// Creates a new "for each" inspector.
func newInspForEach(ctx context.Context, cfg config.Config) (c inspForEach, err error) {
	if err = config.Decode(cfg.Settings, &c); err != nil {
		return inspForEach{}, err
	}

	//  validate option.type
	if !slices.Contains(
		[]string{
			"none",
			"any",
			"all",
		},
		c.Options.Type) {
		return inspForEach{}, fmt.Errorf("condition: for_each: type %q: %v", c.Options.Type, errors.ErrInvalidOption)
	}

	c.inspector, err = NewInspector(ctx, c.Options.Inspector)
	if err != nil {
		return inspForEach{}, fmt.Errorf("condition: for_each: %v", err)
	}

	return c, nil
}

func (c inspForEach) String() string {
	return toString(c)
}

// Inspect evaluates encapsulated data with the Content inspector.
func (c inspForEach) Inspect(ctx context.Context, capsule config.Capsule) (output bool, err error) {
	var results []bool
	for _, res := range capsule.Get(c.Key).Array() {
		tmpCapule := config.NewCapsule()
		tmpCapule.SetData([]byte(res.String()))

		inspected, err := c.inspector.Inspect(ctx, tmpCapule)
		if err != nil {
			return false, fmt.Errorf("condition: for_each: %w", err)
		}
		results = append(results, inspected)
	}

	total := len(results)
	matched := 0
	for _, v := range results {
		if v {
			matched++
		}
	}

	switch c.Options.Type {
	case "any":
		output = matched > 0
	case "all":
		output = total == matched
	case "none":
		output = matched == 0
	default:
		return false, fmt.Errorf("condition: for_each: type %q: %v", c.Options.Type, errors.ErrInvalidOption)
	}

	if c.Negate {
		return !output, nil
	}

	return output, nil
}
