

- Channels(Buffered, just queues): 

    - keep an append only log file for queue insertions, and other log file which just keeps the position of last thing consumed, that's it, use it for disaster recovery or just memory free queue operations

    - endpoints: addToQueue(channel,data) ; popQueue(channel)

- Key value stuff:
    - pure Key value snapshots every once in a while (not reliable + possible data loss)
    - pure WAL, can only be used for recovery, log keeps increasing so it might take forever to recover b doing all operations in the same order
    OR
    - snapshots with WAL
    OR
    - can keep the whole darn WAL of all SETs +
        - will also need to keep the positions(`start_offset + entry_range`) for all keys, that thing needs to be stored somewhere:
            - in memory (breaks the whole point of persistance)
            - in an other file(how to update it ? is it even possible ?)
            - save index snapshots in another file


