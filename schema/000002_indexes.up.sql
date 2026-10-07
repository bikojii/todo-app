CREATE INDEX IF NOT EXISTS users_lists_user_list_idx ON users_lists (user_id, list_id);
CREATE INDEX IF NOT EXISTS lists_items_list_item_idx ON lists_items (list_id, item_id);
CREATE INDEX IF NOT EXISTS lists_items_item_idx ON lists_items (item_id);
