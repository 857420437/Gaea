SHOW VARIABLES;
SHOW STATUS;
SHOW GLOBAL STATUS;
SHOW SESSION STATUS;
SELECT CONNECTION_ID();
SELECT ROW_COUNT();
SELECT GET_LOCK('lock1',10);
SELECT /*+ MAX_EXECUTION_TIME(1000) */ * FROM t1 INNER JOIN t2 where t1.col1 = t2.col1;
(select a.id,a.t_id,a.name,a.pad from test1 a where a.pad=1) union (select c.id,c.o_id,c.name,c.pad from sbtest1.test2 c where c.pad=1) order by id limit 10/*allow_diff_sequence*/
