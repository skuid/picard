alter table data_source_object
	drop constraint data_source_object_data_source_id_foreign,
	add constraint data_source_object_data_source_id_foreign
	foreign key (data_source_id)
	references data_source;