ALTER TABLE isuconp.posts ADD INDEX index_created_at(created_at DESC);
ALTER TABLE isuconp.comments ADD INDEX index_created_at(post_id, created_at DESC);
