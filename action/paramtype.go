package action

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// typeRule holds what one Parameter type means. A new type is one more
// typeRule in paramTypes.
type typeRule interface {
	// fields lists the limit fields the type takes (min, max, pattern, options).
	fields() []string
	// validate checks the limits of a Parameter of this type.
	validate(r *report, at path, p Param)
	// check returns what is wrong with a value in its environment form, as a
	// phrase that follows a subject ("is above the maximum 12"), or "".
	check(p Param, value string) string
	// canonical returns the Parameter with each limit at its default value
	// left out, for the canonical form.
	canonical(p Param) Param
}

var paramTypes = map[ParamType]typeRule{
	TypeString: stringType{},
	TypeInt:    intType{},
	TypeBool:   boolType{},
	TypeChoice: choiceType{},
}

func typeNames() string {
	names := make([]string, 0, len(paramTypes))
	for t := range paramTypes {
		names = append(names, string(t))
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// ---- string

const (
	defaultStringMax = 200
	maxStringMax     = 1024
)

type stringType struct{}

func (stringType) fields() []string { return []string{"max", "pattern"} }

func (stringType) validate(r *report, at path, p Param) {
	if p.Max != nil && (*p.Max < 1 || *p.Max > maxStringMax) {
		r.add(at.key("max"), "max must be from 1 to %d for a string, not %d", maxStringMax, *p.Max)
	}
	if p.Pattern != "" {
		if _, err := compilePattern(p.Pattern); err != nil {
			r.add(at.key("pattern"), "pattern is not a valid RE2 expression: %v", err)
		}
	}
}

func (stringType) check(p Param, v string) string {
	limit := int64(defaultStringMax)
	if p.Max != nil {
		limit = *p.Max
	}
	if problem := textProblem(v, textRule{max: int(limit)}); problem != "" {
		return problem
	}
	if p.Pattern != "" {
		re, err := compilePattern(p.Pattern)
		if err != nil || !re.MatchString(v) {
			return "does not match the pattern " + strconv.Quote(p.Pattern)
		}
	}
	return ""
}

func (stringType) canonical(p Param) Param {
	if p.Max != nil && *p.Max == defaultStringMax {
		p.Max = nil
	}
	return p
}

// compilePattern compiles a pattern so that it matches the whole value.
func compilePattern(pattern string) (*regexp.Regexp, error) {
	if _, err := regexp.Compile(pattern); err != nil {
		return nil, err
	}
	return regexp.Compile(`^(?:` + pattern + `)$`)
}

// ---- int

var decimalPattern = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

type intType struct{}

func (intType) fields() []string { return []string{"min", "max"} }

func (intType) validate(r *report, at path, p Param) {
	if p.Min == nil {
		r.add(at.key("min"), "min is required for an int Parameter")
	}
	if p.Max == nil {
		r.add(at.key("max"), "max is required for an int Parameter")
	}
	if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
		r.add(at.key("min"), "min %d is above max %d", *p.Min, *p.Max)
	}
}

func (intType) check(p Param, v string) string {
	n, problem := parseDecimal(v)
	switch {
	case problem != "":
		return problem
	case p.Min != nil && n < *p.Min:
		return "is below the minimum " + strconv.FormatInt(*p.Min, 10)
	case p.Max != nil && n > *p.Max:
		return "is above the maximum " + strconv.FormatInt(*p.Max, 10)
	}
	return ""
}

func (intType) canonical(p Param) Param { return p }

// parseDecimal reads a whole number written in decimal with no sign but a
// minus, no leading zero and no separators: one spelling for each number.
func parseDecimal(v string) (int64, string) {
	if !decimalPattern.MatchString(v) {
		return 0, "is not a whole number in decimal"
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, "is out of the range of a 64-bit whole number"
	}
	return n, ""
}

// ---- bool

type boolType struct{}

func (boolType) fields() []string                    { return nil }
func (boolType) validate(_ *report, _ path, _ Param) {}
func (boolType) canonical(p Param) Param             { return p }

func (boolType) check(_ Param, v string) string {
	if v != "true" && v != "false" {
		return "is not true or false"
	}
	return ""
}

// ---- choice

const (
	maxOptions = 32
	maxOption  = 60
)

type choiceType struct{}

func (choiceType) fields() []string { return []string{"options"} }

func (choiceType) validate(r *report, at path, p Param) {
	switch {
	case len(p.Options) == 0:
		r.add(at.key("options"), "options is required for a choice Parameter, with at least one option")
		return
	case len(p.Options) > maxOptions:
		r.add(at.key("options"), "a choice has at most %d options, this one has %d", maxOptions, len(p.Options))
	}
	first := map[string]int{}
	for i, o := range p.Options {
		oat := at.key("options").index(i)
		if problem := textProblem(o, textRule{required: true, max: maxOption}); problem != "" {
			r.add(oat, "option %s", problem)
			continue
		}
		if j, used := first[o]; used {
			r.add(oat, "option %q is already listed at %s", o, r.where(at.key("options").index(j)))
			continue
		}
		first[o] = i
	}
}

func (choiceType) check(p Param, v string) string {
	if !slices.Contains(p.Options, v) {
		return "is not one of the options"
	}
	return ""
}

func (choiceType) canonical(p Param) Param { return p }
