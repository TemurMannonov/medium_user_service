CREATE TABLE IF NOT EXISTS permissions(
    id SERIAL PRIMARY KEY,
    user_type VARCHAR CHECK ("user_type" IN('superadmin', 'user')) NOT NULL,
    resource VARCHAR NOT NULL,
    action VARCHAR NOT NULL,
    UNIQUE(user_type, resource, action)   
);

-- users
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'users', 'create');
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'users', 'update');
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'users', 'delete');

INSERT INTO permissions(user_type, resource, action) VALUES ('user', 'users', 'update');
INSERT INTO permissions(user_type, resource, action) VALUES ('user', 'users', 'delete');

-- categories
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'categories', 'create');

-- posts
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'posts', 'create');
INSERT INTO permissions(user_type, resource, action) VALUES ('superadmin', 'posts', 'update');
INSERT INTO permissions(user_type, resource, action) VALUES ('user', 'posts', 'create');
INSERT INTO permissions(user_type, resource, action) VALUES ('user', 'posts', 'update');

