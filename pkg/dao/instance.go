package dao

import (
	"database/sql"
	"time"
)

type Instance struct {
	InstanceID       string `json:"instance_id"`
	ServiceID        string `json:"service_id"`
	InstanceName     string `json:"instance_name"`
	ServiceName      string `json:"service_name"`
	PlanID           string `json:"plan_id"`
	Namespace        string `json:"namespace"`
	OrganizationGUID string `json:"organization_guid"`
	SpaceGUID        string `json:"space_guid"`
	Parameters       string `json:"parameters"`
	Yaml             string `json:"yaml"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

const (
	_insertSQL = `INSERT INTO instances (
			instance_id, 
			service_id, 
			service_name, 
			plan_id, 
			namespace,
			organization_guid,
			space_guid,
			parameters,
			yaml,
			created_at,
			updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`

	_updateSQL = `UPDATE instances SET plan_id = ?, parameters = ?, yaml = ?, updated_at = ? WHERE instance_id = ?`
	_deleteSQL = `DELETE FROM instances WHERE instance_id = ?`
	_selectSQL = `SELECT * FROM instances WHERE instance_id = ?`
)

func (d *Dao) InsertInstance(i *Instance) (int64, error) {
	var res sql.Result
	res, err := d.DB.Exec(_insertSQL, i.InstanceID, i.ServiceID, i.InstanceName,
		i.ServiceName, i.PlanID, i.Namespace, i.OrganizationGUID, i.SpaceGUID, i.Parameters, i.Yaml,
		time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *Dao) UpdateInstance(i *Instance) (int64, error) {
	var res sql.Result
	res, err := d.DB.Exec(_updateSQL, i.PlanID, i.Parameters, i.Yaml,
		time.Now().Format("2006-01-02 15:04:05"), i.InstanceID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *Dao) DeleteInstance(instanceId string) (int64, error) {
	var res sql.Result
	res, err := d.DB.Exec(_deleteSQL, instanceId)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *Dao) SelectInstance(instanceId string) (*Instance, error) {
	res, err := d.DB.Query(_selectSQL, instanceId)
	if err != nil {
		return nil, err
	}

	var instance Instance
	for res.Next() {
		err := res.Scan(&instance.InstanceID, &instance.ServiceID, &instance.InstanceName, &instance.ServiceName,
			&instance.PlanID, &instance.Namespace, &instance.OrganizationGUID, &instance.SpaceGUID,
			&instance.Parameters, &instance.Yaml, &instance.CreatedAt, &instance.UpdatedAt)
		if err != nil {
			return nil, err
		}
	}

	err = res.Err()
	if err != nil {
		return nil, err
	}
	return &instance, nil
}