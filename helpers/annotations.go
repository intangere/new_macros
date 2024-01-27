package helpers

import "github.com/intangere/new_macros/core"

func MatchAny(annotations []core.Annotation, tags []string) []string {
	matched := []string{}
	for _, annotation_set := range annotations {
		for _, annotation_sub_set := range annotation_set.Params {
				for _, annotation := range annotation_sub_set {
				for _, tag := range tags {
					if annotation == tag {
						matched = append(matched, tag)
					}
				}
			}
		}
	}

	return matched
}

func GetTagValues(annotations []core.Annotation, tag string) []string {
	for _, annotation_set := range annotations {
		for _, annotation_sub_set := range annotation_set.Params {
			if annotation_sub_set[0] == tag {
				return annotation_sub_set[1:]
			}
		}
	}
	return []string{}
}

func GetTagValue(annotations []core.Annotation, tag string) (string, bool) {
	// return the first value for a single tag i.e :path="/some/path"->"/some/path"
	for _, annotation_set := range annotations {
		for _, annotation_sub_set := range annotation_set.Params {
			if annotation_sub_set[0] == tag {
				if len(annotation_sub_set) > 1 {
					return annotation_sub_set[1], true
				} else {
					// tag exists but has no value
					return "", true
				}
			}
		}
	}
	return "", false
}
