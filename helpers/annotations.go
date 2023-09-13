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

func GetValues(annotations []core.Annotation, tag string) []string {
	for _, annotation_set := range annotations {
		for _, annotation_sub_set := range annotation_set.Params {
			if annotation_sub_set[0] == tag {
				return annotation_sub_set[1:]
			}
		}
	}
	return []string{}
}
