PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user TEXT NOT NULL UNIQUE,
  pwd TEXT NOT NULL,
  role_id INTEGER NOT NULL,
  exp_time INTEGER NOT NULL,
  flow INTEGER NOT NULL,
  in_flow INTEGER NOT NULL DEFAULT 0,
  out_flow INTEGER NOT NULL DEFAULT 0,
  flow_reset_time INTEGER NOT NULL,
  num INTEGER NOT NULL,
  created_time INTEGER NOT NULL,
  updated_time INTEGER,
  status INTEGER NOT NULL
);

INSERT OR IGNORE INTO user (id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
VALUES (1, 'admin_user', '3c85cdebade1c51cf64ca9f3c09d182d', 0, 2727251700000, 99999, 0, 0, 1, 99999, 1748914865000, NULL, 1);

CREATE TABLE IF NOT EXISTS node (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  secret TEXT NOT NULL,
  ip TEXT,
  server_ip TEXT NOT NULL,
  port_sta INTEGER NOT NULL,
  port_end INTEGER NOT NULL,
  version TEXT,
  http INTEGER NOT NULL DEFAULT 0,
  tls INTEGER NOT NULL DEFAULT 0,
  socks INTEGER NOT NULL DEFAULT 0,
  created_time INTEGER NOT NULL,
  updated_time INTEGER,
  status INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tunnel (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  traffic_ratio REAL NOT NULL DEFAULT 1.0,
  in_node_id INTEGER NOT NULL,
  in_ip TEXT NOT NULL,
  out_node_id INTEGER NOT NULL,
  out_ip TEXT NOT NULL,
  type INTEGER NOT NULL,
  protocol TEXT NOT NULL DEFAULT 'tls',
  flow INTEGER NOT NULL,
  tcp_listen_addr TEXT NOT NULL DEFAULT '[::]',
  udp_listen_addr TEXT NOT NULL DEFAULT '[::]',
  interface_name TEXT,
  created_time INTEGER NOT NULL,
  updated_time INTEGER NOT NULL,
  status INTEGER NOT NULL,
  FOREIGN KEY (in_node_id) REFERENCES node(id) ON DELETE RESTRICT,
  FOREIGN KEY (out_node_id) REFERENCES node(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS speed_limit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  speed INTEGER NOT NULL,
  tunnel_id INTEGER NOT NULL,
  tunnel_name TEXT NOT NULL,
  created_time INTEGER NOT NULL,
  updated_time INTEGER,
  status INTEGER NOT NULL,
  FOREIGN KEY (tunnel_id) REFERENCES tunnel(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS user_tunnel (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  tunnel_id INTEGER NOT NULL,
  speed_id INTEGER,
  num INTEGER NOT NULL,
  flow INTEGER NOT NULL,
  in_flow INTEGER NOT NULL DEFAULT 0,
  out_flow INTEGER NOT NULL DEFAULT 0,
  flow_reset_time INTEGER NOT NULL,
  exp_time INTEGER NOT NULL,
  status INTEGER NOT NULL,
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
  FOREIGN KEY (tunnel_id) REFERENCES tunnel(id) ON DELETE CASCADE,
  FOREIGN KEY (speed_id) REFERENCES speed_limit(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS forward (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  user_name TEXT NOT NULL,
  name TEXT NOT NULL,
  tunnel_id INTEGER NOT NULL,
  in_port INTEGER NOT NULL,
  out_port INTEGER,
  remote_addr TEXT NOT NULL,
  strategy TEXT NOT NULL DEFAULT 'fifo',
  interface_name TEXT,
  in_flow INTEGER NOT NULL DEFAULT 0,
  out_flow INTEGER NOT NULL DEFAULT 0,
  created_time INTEGER NOT NULL,
  updated_time INTEGER NOT NULL,
  status INTEGER NOT NULL,
  inx INTEGER NOT NULL DEFAULT 0,
  lifecycle TEXT NOT NULL DEFAULT 'active',
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
  FOREIGN KEY (tunnel_id) REFERENCES tunnel(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS statistics_flow (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  flow INTEGER NOT NULL,
  total_flow INTEGER NOT NULL,
  time TEXT NOT NULL,
  created_time INTEGER NOT NULL,
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS vite_config (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  value TEXT NOT NULL,
  time INTEGER NOT NULL
);

INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (1, 'app_name', 'flux', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (2, 'captcha_enabled', 'false', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (3, 'captcha_type', 'RANDOM', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (4, 'ip', '', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (5, 'turnstile_enabled', 'false', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (6, 'turnstile_site_key', '', 1755147963000);
INSERT OR IGNORE INTO vite_config (id, name, value, time) VALUES (7, 'turnstile_secret_key', '', 1755147963000);

CREATE TABLE IF NOT EXISTS outbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  retry_count INTEGER NOT NULL DEFAULT 0,
  next_retry_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_forward_user_id ON forward(user_id);
CREATE INDEX IF NOT EXISTS idx_forward_tunnel_id ON forward(tunnel_id);
CREATE INDEX IF NOT EXISTS idx_user_tunnel_user_id ON user_tunnel(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tunnel_tunnel_id ON user_tunnel(tunnel_id);
CREATE INDEX IF NOT EXISTS idx_statistics_flow_user_id ON statistics_flow(user_id);
CREATE INDEX IF NOT EXISTS idx_statistics_flow_created_time ON statistics_flow(created_time);
CREATE INDEX IF NOT EXISTS idx_speed_limit_tunnel_id ON speed_limit(tunnel_id);
