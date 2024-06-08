# analysis
`output_10b_run1.txt`: Results of 10b run1, aka no caching, 2GB memory limit

* After runs 1-5, realized we've been measuring memory on worker and not the GCS. need to change that!!

* Run 6 is flush to disk every 1s using AOF and garbage collection
* Run 7 is the new baseline, don't flush to disk.

run 6 vs run 8 https://chatgpt.com/c/76bd6aef-f711-4067-b626-31446db56773