CREATE TABLE `friends` (
   `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
   `user_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '用户ID',
   `friend_uid` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '好用用户ID',
   `remark` varchar(255) DEFAULT NULL DEFAULT '' COMMENT '好友备注',
   `add_source`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '添加来源',
   `created_at` timestamp NULL DEFAULT NULL COMMENT '添加时间',
   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8  COMMENT='用户好友表';


CREATE TABLE `friend_requests` (
   `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
   `user_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '用户ID',
   `req_uid` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '请求用户ID',
   `req_msg` varchar(255) DEFAULT NULL COMMENT '招呼内容',
   `req_time` timestamp  NOT NULL  COMMENT '请求时间',
   `handle_result`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL,
   `handle_msg` varchar(255) DEFAULT NULL,
   `handled_at`timestamp NULL DEFAULT NULL,
   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8  COMMENT='好友请求表';

 
CREATE TABLE `groups` (
   `id` varchar(24) COLLATE utf8mb4_unicode_ci  NOT NULL COMMENT '主键ID',
   `name` varchar(255) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT ''  COMMENT '群名称',
   `icon` varchar(255) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '群头像图片',
   `status`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL  COMMENT '状态',
   `creator_uid` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL  COMMENT '创建用户ID',
   `group_type` int(11) NOT NULL  COMMENT '群类型',
   `is_verify` boolean NOT NULL  COMMENT '',
   `notification` varchar(255) DEFAULT NULL  COMMENT '群公告',
   `notification_uid` varchar(64) DEFAULT NULL  COMMENT '公告创建用户',
   `created_at` timestamp NULL DEFAULT NULL COMMENT '添加时间',
   `updated_at` timestamp NULL DEFAULT NULL COMMENT '更新时间',
   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '群信息表';

CREATE TABLE `group_members` (
   `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
   `group_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '群ID',
   `user_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '用户ID',
   `role_level`  tinyint COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群角色等级',
   `join_time` timestamp NULL DEFAULT NULL COMMENT '入群时间',
   `join_source`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '入群来源方式',
   `inviter_uid` varchar(64) DEFAULT NULL  COMMENT '入群邀请人',
   `operator_uid` varchar(64) DEFAULT NULL  COMMENT '操作人',
   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8  COMMENT '群成员信息表';

CREATE TABLE `group_requests` (
   `id` int(11) unsigned NOT NULL AUTO_INCREMENT  COMMENT '主键ID',
   `req_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '' COMMENT '请求ID',
   `group_id` varchar(64) COLLATE utf8mb4_unicode_ci  NOT NULL  DEFAULT '' COMMENT '群ID',
   `req_msg` varchar(255) DEFAULT NULL COMMENT '招呼内容',
   `req_time` timestamp NULL DEFAULT NULL COMMENT '请求时间',
   `join_source`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '添加来源',
   `inviter_user_id` varchar(64) DEFAULT NULL COMMENT '要求用户ID',
   `handle_user_id` varchar(64) DEFAULT NULL COMMENT '入群处理人ID',
   `handle_time` timestamp NULL DEFAULT NULL COMMENT '处理时间',
   `handle_result`  tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '处理结果',
   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8  COMMENT '入群请求表';