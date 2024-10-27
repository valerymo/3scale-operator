package helper

import (
	v1 "k8s.io/api/core/v1"
	apimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ApimanagerWatchedBySelectorKey = "apimanager.apps.3scale.net/watched-by"
)

type DataChangedPredicate struct {
	predicate.Funcs
}

func (d DataChangedPredicate) Update(e event.UpdateEvent) bool {
	oldSecret := e.ObjectOld.(*v1.Secret)
	newSecret := e.ObjectNew.(*v1.Secret)

	// Compare the data fields of the old and new Secret
	return !reflect.DeepEqual(oldSecret.Data, newSecret.Data)
}

func (d DataChangedPredicate) Create(e event.CreateEvent) bool {
	return true // Trigger on creation
}

type BothPredicate struct {
	WatchedByLabelPredicate predicate.Predicate
	DataPredicate           predicate.Predicate
}

func (b BothPredicate) Create(e event.CreateEvent) bool {
	return b.WatchedByLabelPredicate.Create(e) && b.DataPredicate.Create(e)
}

func (b BothPredicate) Update(e event.UpdateEvent) bool {
	return b.WatchedByLabelPredicate.Update(e) && b.DataPredicate.Update(e)
}

func (b BothPredicate) Delete(e event.DeleteEvent) bool {
	return b.WatchedByLabelPredicate.Delete(e) && b.DataPredicate.Delete(e)
}

func (b BothPredicate) Generic(e event.GenericEvent) bool {
	return b.WatchedByLabelPredicate.Generic(e) && b.DataPredicate.Generic(e)
}

func GetWatchedByDataOnlySecretLabelPredicate(selectorValue string) predicate.Predicate {
	// create Predicate for Watch to trigger only when Data changed in redis secret
	// "BothPredicate" - combines WatchedByLabel and Data predicates
	redisSystemWatchedByLabelPredicate, err := predicate.LabelSelectorPredicate(apimachinerymetav1.LabelSelector{
		MatchLabels: map[string]string{
			ApimanagerWatchedBySelectorKey: selectorValue,
		},
	})
	if err != nil {
		return nil
	}
	dataChangedPredicate := DataChangedPredicate{}
	bothPredicate := BothPredicate{
		WatchedByLabelPredicate: redisSystemWatchedByLabelPredicate,
		DataPredicate:           dataChangedPredicate,
	}
	return bothPredicate
}
