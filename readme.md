

- Channels(Buffered, just queues): 

    - keep an append only log file for queue insertions, and other log file which just keeps the position of last thing consumed, that's it, use it for disaster recovery or just memory free queue operations

    - endpoints: PushQueue(channel,data) ; PopQueue(channel)

- Key value stuff:
    - pure Key value snapshots every once in a while (not reliable + possible data loss) -> not chosen
    - pure WAL, can only be used for recovery, log keeps increasing so it might take forever to recover b doing all operations in the same order -> not chosen
    OR
    - state snapshots with WAL. endpoints: SetVal(key, value) ,->chosen, we do this for now
    OR
    - can keep the whole darn WAL of all SETs +
        - will also need to keep the positions(`start_offset + entry_range`) for all keys, that thing needs to be stored somewhere:
            - in memory (breaks the whole point of persistance)
            - in an other file(how to update it ? is it even possible ?)
            - save index snapshots in another file
        - -> not chosen



store all needed files in home_folder/.persisto/

each project we import this in needs a project UUID, everything for that project is stored in `PROJECT_UUID_{}`

queue operations: each queue has its own file(`queue_name_queue_{}.log`), for each queue we have two files:
    - `Ins` file: just a WAL log for insertions
    - `Ins_index` file: keep the range for each entry, follow a certain format -> `/s{start_offset,data_range}/e ....`, make sure to keep a space character between each entry in here
    - how to keep track of the processed stuff ? by adding another file ? keep appending stuff to it ? 
    - `processed` file: when some queue element gets processed, we update it here, essentially by appending the `/s{start_offset,data_range}/e` of that thing in this file

map operations:
    - `WAL` file: raw entries of all set operations
    - `INCREMENTAL_NUM_SNAP` snapshot of all the keys in the system and which range their values are in in the `WAL` file, eg. `key`: `{start_offset,data_range}`, we maintain this in memory and once every while, we make a snapshot of it and put it in a new file with incremental prefix
    - `processed` file: just like queues, but we only append the positions for which a snapshot has been taken

to restore:
    (when a program boots up, it checks for unprocessed stuff, maybe it crashed last time and has some stuff left to do, so it essentially tries to restore what it has left)
    common stuff for both queue and maps: `project uuid`,
    - queues: (eg file for one queue: `Ins file`: `{project_uuid}_{queue_name}_queue_Ins.log`, `Ins_index file`: `{project_uuid}_{queue_name}_queue_Ins_index.log`, `processed file`: `{project_uuid}_{queue_name}_queue_processed.log`), to restore, we just read the last thing processed_file processed by reading backwards, no need to read the whole file
    - map: just take the latest snapshot,restore it and if the position of the last entry in `processed` file is earlier than the `WAL` file size, we start from that position in `WAL` file and do the operations in that order untill we get it


only one process touches these files at a time

about the `processed` log, we only read it backwards to find the last entry, nothing else, so we don't need to read it all

we can add some sort of versioning to `processed` files so that if due to application crash, the old processed file remains even after a snapshot was created, we ignore it

ignore infinite log growth issue for now






























need folder_path for the project intergrating persisto

keep all files by creating a directory in that folder `.persisto` and the names of files would just be the names of variables


persisto maintains some internal use KV pairs for its working, the internal keys have `_` as prefix

the below stuff ignores WAL as it is internally managed

## internal implementation of KV store:

two files: main and index

- have an in memory key index pair
- append whatever write operation comes to main and append the updated {key,new_offset} in an index file
- update in memory index

startup step: 
    - populate in memory index by looking at index file , if no index file then read whole main

## internal implementation of queue:

files: main , one KV pair

- append whatever thing in main
- an read_offset KV pair which keeps track of last thing consumed

- to consume an entry from queue, read the entry and update the read_offset





ENSURE ALL YOUR STRUCTS YOU WANNA USE ARE REGISTERED WITH GOB SOMEWHERE LIKE THIS: `gob.Register(stupidStruct{})`