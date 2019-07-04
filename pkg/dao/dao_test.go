package dao

import (
	"fmt"
	"testing"
)

func TestDao(t *testing.T) {

	c := &Config{
		Addr:     "127.0.0.1",
		Port:     "3306",
		UserName: "root",
		Password: "root",
		DB:       "servicebroker",
		Active:   100,
		Idle:     10,
	}

	d, err := New(c)
	if err != nil {
		t.Fatal(err)
	}

	i := &Instance{
		InstanceID:       "a",
		SpaceGUID:        "a",
		ServiceName:      "a",
		ServiceID:        "a",
		OrganizationGUID: "a",
		PlanID:           "c",
		Parameters:       "d",
	}

	_, err = d.InsertInstance(i)
	if err != nil {
		t.Fatal(err)
	}

	ii, err := d.SelectInstance("a")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(ii)

	i = &Instance{
		InstanceID:       "a",
		SpaceGUID:        "a",
		ServiceName:      "a",
		ServiceID:        "a",
		OrganizationGUID: "a",
		PlanID:           "dddddddddd",
		Parameters:       "dddddddddd",
	}

	_, err = d.UpdateInstance(i)
	if err != nil {
		t.Fatal(err)
	}

	ii, err = d.SelectInstance("a")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(ii)

	_, err = d.DeleteInstance("a")
	if err != nil {
		t.Fatal(err)
	}
}