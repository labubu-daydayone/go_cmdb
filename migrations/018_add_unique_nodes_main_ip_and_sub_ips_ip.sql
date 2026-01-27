-- Add unique constraint on nodes.main_ip
ALTER TABLE nodes ADD UNIQUE KEY uk_nodes_main_ip (main_ip);

-- Add unique constraint on node_sub_ips.ip
ALTER TABLE node_sub_ips ADD UNIQUE KEY uk_node_sub_ips_ip (ip);
