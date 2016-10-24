CREATE TABLE IF NOT EXISTS `settings` (
    id BIGINT auto_increment PRIMARY KEY,
    skey VARCHAR(255),
    svalue TEXT
);
CREATE TABLE IF NOT EXISTS `portfolios` (
    id BIGINT NOT NULL auto_increment PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    value int
);
CREATE TABLE IF NOT EXISTS `notes` (
    id BIGINT auto_increment PRIMARY KEY,
    value TEXT,
    portfolio_id BIGINT NOT NULL,
    position_id BIGINT NOT NULL
);
CREATE TABLE IF NOT EXISTS `positions` (
    id BIGINT auto_increment PRIMARY KEY,
    portfolio_id BIGINT NOT NULL,
    started_at DATETIME,
    closed_at DATETIME,
    ptype VARCHAR(255),
    buy int,
    sell int,
    stop_loss int,
    quantity int
);
CREATE TABLE IF NOT EXISTS `plays` (
    id BIGINT auto_increment PRIMARY KEY,
    position_id BIGINT NOT NULL,
    day DATETIME,
    open INT,
    high INT,
    low INT,
    pvolume INT,
    pchange INT,
    pchange_percent INT,
    adj_close INT,
    data_source VARCHAR(255)
);