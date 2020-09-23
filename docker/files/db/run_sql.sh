#/bin/bash -x

pg_ctl start -D /database
for file in $*
do 
  echo "Running sqlfile $file."
  psql $DB_FLAG -f $file postgres
  err=$?
  if [  $err -ne 0 ]
  then
		echo "$file failed to execute"
		exit $err
  fi
done
pg_ctl stop -D /database

