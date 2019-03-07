/*

 Copyright 2019 The KubeSphere Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.

*/
package resources

import (
	"sort"
	"strings"

	lister "k8s.io/client-go/listers/batch/v2alpha1"

	"k8s.io/api/batch/v2alpha1"
	"k8s.io/apimachinery/pkg/labels"
)

type cronJobSearcher struct {
	cronJobLister lister.CronJobLister
}

func cronJobStatus(item *v2alpha1.CronJob) string {
	if item.Spec.Suspend != nil && *item.Spec.Suspend {
		return paused
	}
	return running
}

// Exactly match
func (*cronJobSearcher) match(match map[string]string, item *v2alpha1.CronJob) bool {
	for k, v := range match {
		switch k {
		case status:
			if cronJobStatus(item) != v {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (*cronJobSearcher) fuzzy(fuzzy map[string]string, item *v2alpha1.CronJob) bool {

	for k, v := range fuzzy {
		switch k {
		case name:
			if !strings.Contains(item.Name, v) && !strings.Contains(item.Labels[displayName], v) {
				return false
			}
		case label:
			if !searchFuzzy(item.Labels, "", v) {
				return false
			}
		case annotation:
			if !searchFuzzy(item.Annotations, "", v) {
				return false
			}
			return false
		case app:
			if !strings.Contains(item.Labels[chart], v) && !strings.Contains(item.Labels[release], v) {
				return false
			}
		case keyword:
			if !strings.Contains(item.Name, v) && !searchFuzzy(item.Labels, "", v) && !searchFuzzy(item.Annotations, "", v) {
				return false
			}
		default:
			if !searchFuzzy(item.Labels, k, v) && !searchFuzzy(item.Annotations, k, v) {
				return false
			}
		}
	}

	return true
}

func (*cronJobSearcher) compare(a, b *v2alpha1.CronJob, orderBy string) bool {
	switch orderBy {
	case createTime:
		return a.CreationTimestamp.Time.After(b.CreationTimestamp.Time)
	case name:
		fallthrough
	default:
		return strings.Compare(a.Name, b.Name) <= 0
	}
}

func (s *cronJobSearcher) search(namespace string, conditions *conditions, orderBy string, reverse bool) ([]interface{}, error) {
	cronJobs, err := s.cronJobLister.CronJobs(namespace).List(labels.Everything())

	if err != nil {
		return nil, err
	}

	result := make([]*v2alpha1.CronJob, 0)

	if len(conditions.match) == 0 && len(conditions.fuzzy) == 0 {
		result = cronJobs
	} else {
		for _, item := range cronJobs {
			if s.match(conditions.match, item) && s.fuzzy(conditions.fuzzy, item) {
				result = append(result, item)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if reverse {
			tmp := i
			i = j
			j = tmp
		}
		return s.compare(result[i], result[j], orderBy)
	})

	r := make([]interface{}, 0)
	for _, i := range result {
		r = append(r, i)
	}
	return r, nil
}