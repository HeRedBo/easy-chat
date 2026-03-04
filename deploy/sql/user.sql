CREATE TABLE `users` (
     `id` varchar(24) COLLATE utf8mb4_unicode_ci  NOT NULL COMMENT '主键ID',
     `avatar` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '用户头像',
     `nickname` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '用户昵称',
     `phone` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT  '' COMMENT '手机号',
     `password` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '密码',
     `status` tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '状态',
     `sex` tinyint COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '性别',
     `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
     `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
--      `created_at` timestamp NULL DEFAULT NULL  COMMENT '创建时间',
--      `updated_at` timestamp NULL DEFAULT NULL COMMENT '更新时间',,
     PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci  COMMENT='用户表';
