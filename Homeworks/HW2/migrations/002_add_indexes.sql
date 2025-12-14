CREATE INDEX idx_images_created_at ON images(created_at);
CREATE INDEX idx_images_format_created ON images(format, created_at);
CREATE INDEX idx_images_width ON images(width);
CREATE INDEX idx_images_height ON images(height);

CREATE INDEX idx_images_url_prefix ON images(url(255));
CREATE INDEX idx_images_filename_prefix ON images(filename(255));
