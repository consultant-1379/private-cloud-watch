# Understanding and using ha-state.py output

The output has 4 parts, each providing potentially complimentary information
about HA events in the logs.  This file explains some ways the different
sections are useful, followed by some example output.

## Sections of ha-state output

### 1. A header with time information.

It lists both the span of time covered by the input logs, and the period of
time between the first and last HA event noticed.

### 2. First histogram, with each line showing a 30 minute bucket.

Buckets with no HA events are not displayed to keep the output compact, and
effectively zero length if no events occurred. To help see any contiguous
buckets with events, time gaps are demarked on the NEXT line with a "GAP >>"
tag and associated time gap.

In the example Time Histogram below, you can see there was a 1 hour period with
at least 15 HA events. Note that this view also groups events by VM, showing
the total number of VM's in bar, and the most frequently occuring VM, and its
occurrence count. In the example, there is no single VM is dominating, telling
us that the outage was probably not due to a problem with a specific VM type.

### Part 3. Second histogram, showing HA events for each vm name.

Here you can check in more detail if specific VM's are having more problems
than others. Here the events are grouped by HA id's. If the TopGroup has a
significant number of events compared to the total events in the current
bucket, then a single HA event was retrying frequently. This contrasts from a
VM having problems addressed by _different_ HA events, and indicates a
different path for investigation. (By HA-id vs by VM-name)

The last part of the line, after the "#Evts" field, shows the N events that
occurred most closely together for the current line. They are ordered by time
of occurrence. You can quicky scan this information to see if there are one or
more very close events. This may indicte a time period that was particularly
volatile for this vm type.

### 4. Third histogram, showing HA events by HA ID.

This is nearly the inverse of the previous histogram. It is useful to see if
one or more HA id's had multiple events.  If so, this indicates HA Workflows
were not able to complete successfully and had to retry on the SAME HA id
multiple times.

Grouping is by VM name. Check if any vm is causing a disproportionate amount
of the events for given HA id.

## Example output

1. Header
```
     ========================== HA Workflow REPORT ============================
     Log Span                   2019-07-18 01:12:35.496000 2019-08-19 08:46:51.165000
     Log Elapsed time           32 days, 7:34:15.669000
     EventActivity Span          2019-08-03 00:06:44.788000 2019-08-19 08:45:12.981000
     EventActivity Elapsed time  16 days, 8:38:28.193000
```

2. Time bucketed Histogram
```
     ==== Time Histogram, 30 minutes per bucket
     ====   Note: 0 occurrence buckets not displayed
     key             Occurrences- Grps TopGroup
     2019-08-03 00:00 +            1    ('tbaytelenm01-comecimpolicy-1', 1)
    									 Gap >>  5 days, 23:30:00
     2019-08-08 23:30 +            1    ('tbaytelenm01-comecimpolicy-0', 1)
   										Gap >>  7 days, 22:30:00
     2019-08-16 22:00 ++++++++++   10   ('tbaytelenm01-wpserv-0', 1)
     2019-08-16 22:30 +++++        5    ('tbaytelenm01-cmserv-1', 1)
    									 Gap >>  1:30:00
     2019-08-17 00:00 +            1    ('tbaytelenm01-emp-0', 1)
```

3. VM bucketed Histogram
```
     ====  # of DELETE_COMPLETE's per vm ordered by frequency
     key                            Occurrences- ----First-Occur---- -------Span-------- Grps TopGroup & #Evts      Timespan between Top N nearest events.
     tbaytelenm01-fmx-1             +++++        2019-08-16 22:34:07     1 day, 22:22:45 5    1566033004598 1 (First:2019-08-16 22:34:07)| 6:37:38| 0:12:39| 0:21:27| 1 day, 15:10:57
     tbaytelenm01-vaultserv-1       +++++        2019-08-17 04:38:26     2 days, 1:23:29 5    1566032914420 1 (First:2019-08-17 04:38:26)| 0:32:46| 0:31:13| 1 day, 15:14:28| 9:04:59

```
4. HA Key bucketed Histogram
```
     ==== HA id's by frequency
     key                            Occurrences- ----First-Occur---- Span    Grps TopGroup
     1566107096120                  +++++++++++  2019-08-18 01:46:46 0:00:41 11   ('tbaytelenm01-scp-1', 1)
     1566007864175                  ++++++++++   2019-08-16 22:12:53 0:00:20 10   ('tbaytelenm01-wpserv-0', 1)
     1566078716108                  ++++++++     2019-08-17 17:56:09 3:05:30 8    ('tbaytelenm01-comecimpolicy-1', 1)
     1566030964811                  ++++++++     2019-08-17 04:38:22 0:16:51 7    ('tbaytelenm01-itservices-1', 2)
```


## Caveats

ha-state.py may not capture all HA events, as there are cases where no
"DELETE_COMPLETE" is output to the logs. This is not currently viewed as a
problem since empirical evidence shows that instability and service outages
have enough HA events that the event is easily visible in he histograms.

To make the tool more accurate will require a close reading of the enm source
code to learn all possible HA workflow states, and their corresponding log
outputs.

## Searching for vm's, HA ids, etc in logs

ha-state.py can provide an idea about what tenant(s) experienced HA events, and
even what time frame and a smple of the vm's involved.  This alone might be
enough to act on, but often you'll want to dig further in the Jboss logs using
the output from ha-state.py as the starting point(s).

Often, you'll take a HA id and/or VM name in the time period of interest,
provided by ha-state.py, and search the logs for it, to find related log
information in same time period.

You can scan for the HA id manually using less, or sample using something like:

```grep -C 10 [HA-ID] [file]```

## Runnning ha-state.py standalone, instead of viewing pre-generated reports

You can run ha-state.py on any jboss logs by either piping the logs to stdin:

```zcat newlogs/Tbytel/server*.gz | /opt/enmscout/bin/ha-state.py ```

or by listing them at the command line:

```/opt/enmscout/bin/ha-state.py log1.gz log2.gz log3.gz ...```
