DROP TABLE IF EXISTS `data`;
CREATE TABLE `data` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `uuid` char(36) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `timestamp` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `bridge_id` char(12) NOT NULL,
  `sensor` varchar(16) DEFAULT NULL,
  `battery` varchar(16) DEFAULT NULL,
  `signal_rssi` smallint(6) DEFAULT NULL,
  `mt` varchar(16) DEFAULT NULL,
  `temperature_c` decimal(4,2) DEFAULT NULL,
  `humidity` decimal(3,1) DEFAULT NULL,
  `wind_kmh` decimal(5,2) DEFAULT NULL,
  `wind_direction` decimal(4,1) DEFAULT NULL,
  `rainfall_mm` decimal(7,3) DEFAULT NULL,
  `pressure_pa` int(11) DEFAULT NULL,
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `data_uuid_idx` (`uuid`),
  KEY `data_timestamp_idx` (`timestamp`),
  KEY `data_bridge_id_idx` (`bridge_id`),
  KEY `data_timestamp_bridge_id_idx` (`timestamp`,`bridge_id`)
) ENGINE=InnoDB AUTO_INCREMENT=130100 DEFAULT CHARSET=latin1;
