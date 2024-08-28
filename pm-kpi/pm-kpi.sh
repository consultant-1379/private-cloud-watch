#!/bin/bash

DDC_DATA_PATH=/var/ericsson/ddc_data
DEPLOYMENT_NAME=$(hostname | awk -F '-' '{print $1}')
FOLDER_NAME=$(date +"%d%m%y")
BEAN_NAME=FileCollectionStatistics # Metric to collect
CURRENT_DATETIME=$(date +"%d-%m-%y %T.%3N")
FILE_NAME=pm-kpi.txt # File name to store last colelct date
PM0_COLLECTION_SUCCESS=true
PM1_COLLECTION_SUCCESS=true

# Get time of the record we read in the last run if the file exists, otherwise create it.
if [ -f "$FILE_NAME" ]; then
    PM0_LAST_COLLECT=$(head -n 1 $FILE_NAME)
    PM1_LAST_COLLECT=$(tail -n 1 $FILE_NAME)
else
    touch $FILE_NAME
fi

# Get the last recrod from the file.
LAST_COLLECTION_DATA_PMSERVE_0=$(grep "$BEAN_NAME" "$DDC_DATA_PATH/$DEPLOYMENT_NAME-pmserv-0_TOR/$FOLDER_NAME/instr.txt" | tail -n 1)
LAST_COLLECTION_DATA_PMSERVE_1=$(grep "$BEAN_NAME" "$DDC_DATA_PATH/$DEPLOYMENT_NAME-pmserv-1_TOR/$FOLDER_NAME/instr.txt" | tail -n 1)

# If time of last run is that same as time from this run, DDC has not updated the value.
PM0_TIME=$(echo $LAST_COLLECTION_DATA_PMSERVE_0 | awk -F ' ' '{print $1,$2}')
if [ "$PM0_LAST_COLLECT" != "$PM0_TIME" ]; then # If it is different, get the success and fail values from the file
    PM0_SUCCESS=$(echo $LAST_COLLECTION_DATA_PMSERVE_0 | awk -F ' ' '{print $8}')
    PM0_FAIL=$(echo $LAST_COLLECTION_DATA_PMSERVE_0 | awk -F ' ' '{print $9}')
    echo $PM0_TIME >$FILE_NAME
else
    PM0_COLLECTION_SUCCESS=false # set flag to false if time not apdated
fi

PM1_TIME=$(echo $LAST_COLLECTION_DATA_PMSERVE_1 | awk -F ' ' '{print $1,$2}')
if [ "$PM0_LAST_COLLECT" != "$PM0_TIME" ]; then
    PM1_SUCCESS=$(echo $LAST_COLLECTION_DATA_PMSERVE_1 | awk -F ' ' '{print $8}')
    PM1_FAIL=$(echo $LAST_COLLECTION_DATA_PMSERVE_1 | awk -F ' ' '{print $9}')
    echo $PM1_TIME >>$FILE_NAME
else
    PM1_COLLECTION_SUCCESS=false
fi

# If both flags are false, no new data was collected by DDC for this metric,
# We assume this is a DDC fault and return 0. This could just be caused by the script being called too often. The file is updated every minute.
if [ $PM0_COLLECTION_SUCCESS != "true" ] && [ $PM1_COLLECTION_SUCCESS != "true" ]; then
    echo 0
    exit
fi

TOTAL_SUCCESS=$(($PM0_SUCCESS + $PM1_SUCCESS))
TOTAL_FAIL=$(($PM0_FAIL + $PM1_FAIL))
TOTAL=$(($TOTAL_SUCCESS + $TOTAL_FAIL))

if [ "$TOTAL" -ne 0 ]; then
    PERCENT_SUCCESS=$(awk "BEGIN{print($TOTAL_SUCCESS/$TOTAL*100)}")
else
    # If total is zero, there are no subscribed nodes. Always send 100 as there is nothing to collect.
    PERCENT_SUCCESS=100
fi

echo $PERCENT_SUCCESS
