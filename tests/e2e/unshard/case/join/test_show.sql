show columns in t1;
show columns from t1;
show full columns from t1;
show columns from t1 from sbtest1/*not_support*/;
-- show columns from t1 from sbtest1_0;
show full columns from t1 from sbtest1;
show full columns from t1 from sbtest1 like 'n%';
show full columns from t1 from sbtest1 where field like 's%';
show table status/*allow_diff*/;
show table status like 'aly_o%'/*allow_diff*/;
show tables;
show full tables;
show tables from sbtest1;
show tables in sbtest1;
show full tables from sbtest1;
show full tables in sbtest1;
show tables like 'aly_o%';
show full tables from sbtest1 like 'aly%';
show full tables in sbtest1 where table_type like 'base%';
#create INDEX index_001 ON t1 (ID);
show index from t1;
show index in t1;
show index from t1 from sbtest1;
show index in t1 in sbtest1;
show index in t1 from sbtest1;
show index from t1 in sbtest1;
show keys from t1;
show keys in t1;
show keys from t1 from sbtest1;
show keys from t1 in sbtest1;
show keys in t1 in sbtest1;
show keys in t1 from sbtest1;
#drop index index_001 on t1;
create database  if not exists sbtest1;
show create database sbtest1;
show create schema sbtest1;
show create schema if not exists sbtest1;
show create database if not exists sbtest1;
show databases;
show schemas;
show databases like 'sbtest1';
show schemas like 'sbtest1';
show open tables;
show open tables from sbtest1;
show open tables in sbtest1;
show open tables from sbtest1 like 'aly_o%';
