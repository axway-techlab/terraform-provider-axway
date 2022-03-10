package axwayapi

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	limit_pattern = `^\s*(?P<nb>\d+) *(?P<unit>(MB)|(msg)) *((per)|(\/)) *(?P<t>\d*) *(?P<tunit>(se?c?o?n?d?)|(mi?n?u?t?e?)|(ho?u?r?)|(da?y?)|(we?e?k?))s?$`
)

var rLimit = regexp.MustCompile(limit_pattern)

var TFQuotaSchema = schemaMap{
	"name":        optional(_string()),
	"description": optional(_string(), " "),
	"type":        readonly(_string()),
	"system":      readonly(_bool()),
	"restriction": required(_setMin(1, TFRestriction)),
}
var TFRestriction = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"api_id": required(_string()),
		"method": desc(optional(_string(), "*"), `the name of the method to limit, or '*' (the default) to limit them all`),
		"limit": desc(required(_apply(compressRestriction, _string(r(limit_pattern)))),
			`A string in the form :
			  - <nb_msg> msg per <t> <tunit>
			  - <nb_mb> MB per <t> <tunit>
			 In both cases, <t> is a positive int, <tunit> is one of second, minute, hour, day, week (with an optional trailing 's')
			 If <t> is missing, a value of 1 is implied
			 The first form places a limit in the nb od messages per interval.
			 Conversely, the second form places a limit in mega bytes in the interval.
			 Example:
			 	- 20 MB per minute
				- 100 msg per 2 minutes
			The syntax is pretty lax, so "10MB/s", "10msg per 5secs", "10 msg / 5 sec" all work.
			`),
	},
}

func compressRestriction(s interface{}) string {
	r := RestrictionFromString(s.(string))
	return fmt.Sprintf("%d%s/%d%s", r.nb, r.unit, r.time, canonTUnit(r.tunit))
}

type restriction struct {
	nb, time    int
	unit, tunit string
}

func RestrictionFromString(s string) *restriction {
	matches := rLimit.FindStringSubmatch(s)
	r := &restriction{time: 1}
	nb, err := strconv.Atoi(matches[rLimit.SubexpIndex("nb")])
	if err != nil {
		panic(err)
	}
	r.nb = nb

	var t int
	_t := matches[rLimit.SubexpIndex("t")]
	if _t == "" {
		t = 1
	} else {
		t, err = strconv.Atoi(_t)
		if err != nil {
			panic(err)
		}
	}
	r.time = t
	r.unit = matches[rLimit.SubexpIndex("unit")]
	r.tunit = matches[rLimit.SubexpIndex("tunit")]
	return r
}

func flattenRestriction(restriction []client.Constraint) *schema.Set { //[]flattenMap {
	ret := make([]interface{}, 0)
	for _, r := range restriction {
		ret = append(ret, flattenMap{
			"api_id": r.Api,
			"method": r.Method,
			"limit":  flattenRestrictionConfig(r.Config),
		})
	}

	return schema.NewSet(func(i interface{}) int {
		f := fnv.New32a()
		f.Write([]byte(fmt.Sprintf("%#+v", i)))
		return int(f.Sum32())
	}, ret)
}
func flattenRestrictionConfig(config interface{}) string {
	switch c := config.(type) {
	case client.ConstraintConfigMb:
		return fmt.Sprintf("%dMB/%d%s", c.Mb, c.Per, canonTUnit(c.Period))
	case client.ConstraintConfigMsg:
		return fmt.Sprintf("%dmsg/%d%s", c.Msg, c.Per, canonTUnit(c.Period))
	default:
		panic(fmt.Errorf("cannot parse config (unknown type %T): %#+v", config, config))
	}
}
func expandRestrictions(v interface{}) (quota []client.Constraint) {
	c := v.(*schema.Set).List()
	r := make([]client.Constraint, 0)
	for _, b := range c {
		a := b.(map[string]interface{})
		if a["api_id"] != "" {
			// Unfortunate test but this seems to be necessary
			// to avoid phantom items in the set
			c := expandRestrictionConfig(a["limit"])
			var cc client.Constraint
			switch c.(type) {
			case client.ConstraintConfigMb:
				cc.Config = c
				cc.Type = "throttlemb"
			case client.ConstraintConfigMsg:
				cc.Config = c
				cc.Type = "throttle"
			default:
				panic(fmt.Errorf("cannot understand this restriction: %#+v", c))
			}
			cc.Api = a["api_id"].(string)
			cc.Method = a["method"].(string)
			r = append(r, cc)
		}
	}
	return r
}

func expandRestrictionConfig(v interface{}) (config interface{}) {
	matches := rLimit.FindStringSubmatch(v.(string))
	if len(matches) == 0 {
		panic(fmt.Errorf("cannot understand the restriction.limit string: '%+#v'", v))
	}
	nb, err := strconv.Atoi(matches[rLimit.SubexpIndex("nb")])
	if err != nil {
		panic(err)
	}
	t, err := strconv.Atoi(matches[rLimit.SubexpIndex("t")])
	if err != nil {
		panic(err)
	}
	unit := matches[rLimit.SubexpIndex("unit")]
	tunit := canonTUnit(matches[rLimit.SubexpIndex("tunit")])
	switch unit {
	case "MB":
		return client.ConstraintConfigMb{Mb: nb, Per: t, Period: tunit}
	case "msg":
		return client.ConstraintConfigMsg{Msg: nb, Per: t, Period: tunit}
	}
	return r
}

func canonTUnit(tunit string) string {
	units := make(map[string]*regexp.Regexp, 5)
	units["second"] = regexp.MustCompile(`^se?c?o?n?d?s?$`)
	units["minute"] = regexp.MustCompile(`^mi?n?u?t?e?s?$`)
	units["hour"] = regexp.MustCompile(`^ho?u?r?s?$`)
	units["day"] = regexp.MustCompile(`^da?y?s?$`)
	units["week"] = regexp.MustCompile(`^we?e?k?s?$`)

	// Works reliably only bc patterns do not overlap.
	// If for instance, "month" is added, it would conflict with "minutes"
	// since both would match a single 'm'.
	for norm, pattern := range units {
		if pattern.MatchString(tunit) {
			return norm
		}
	}
	panic(fmt.Errorf("unnknown time unit '%s'", tunit))
}
