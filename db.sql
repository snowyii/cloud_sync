CREATE TABLE `POST` (
  `uuid` char(36) NOT NULL,
  `author` varchar(64) DEFAULT NULL,
  `message` varchar(1024) DEFAULT NULL,
  `likes` int(10) unsigned DEFAULT 0,
  `del` tinyint(4) DEFAULT 0,
  `last_update` int(11) NOT NULL,
  `image` tinyint(4) DEFAULT 0,
  `img_last_update` int(11) NOT NULL,
  PRIMARY KEY (`uuid`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


CREATE INDEX post_index ON `POST` (`last_update`) ;