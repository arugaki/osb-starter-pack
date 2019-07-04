CREATE DATABASE `servicebroker` CHARACTER SET utf8 COLLATE utf8_general_ci;

use servicebroker;

CREATE TABLE IF NOT EXISTS `instances`(
   `instance_id` VARCHAR(100) NOT NULL COMMENT '服务实例ID',
   `instance_name` VARCHAR(100) NOT NULL COMMENT '服务实例名',
   `service_id` VARCHAR(100) NOT NULL COMMENT '服务ID',
   `service_name` VARCHAR(100) NOT NULL COMMENT '服务名',
   `plan_id` VARCHAR(100) NOT NULL COMMENT '服务规格ID',
   `namesapce` VARCHAR(100) NOT NULL COMMENT 'Namspace名',
   `organization_guid` VARCHAR(100) NOT NULL COMMENT '组织ID',
   `space_guid` VARCHAR(100) NOT NULL COMMENT '空间ID',
   `parameters` TEXT NOT NULL COMMENT '服务创建等操作所需填写的参数',
   `yaml` TEXT NOT NULL COMMENT '部署服务的kubernetes编排文件',
   `created_at` VARCHAR(50) COMMENT '创建时间',
   `updated_at` VARCHAR(50) COMMENT '更新时间',
   PRIMARY KEY ( `instance_id` )
) ENGINE=InnoDB DEFAULT CHARSET=utf8;