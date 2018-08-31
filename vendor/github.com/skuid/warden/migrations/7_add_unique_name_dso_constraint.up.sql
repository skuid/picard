ALTER TABLE data_source_object
	ADD CONSTRAINT data_source_object_name_data_source_id_unique
	unique (name, data_source_id);
ALTER TABLE public.data_source_object ADD table_name VARCHAR(128) NULL;