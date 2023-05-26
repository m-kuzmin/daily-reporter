# Daily Reporter

This is a telegram bot that can generate a daily report in a telegram chat. This report comes from a GitHub project and is roughly structured like this:

> **Daily Report [Date]:**
>
> **Today I worked on:**
> - Task 1
> - Task 2
> - Task 3
>
> **Tomorrow I will be working on:**
> - Task 4
> - Task 5
> - Task 6
>
> **Discovery of the Day:**
> Today, I discovered ...
>
> **Questions/Blockers:**
> - Question 1
> - Question 2
> - Blocker 1

# Workflow using the bot

The bot gets the lists by collecting items by status from a GitHub project. The labels are: 
- *In Progress* - Tomorrow's todo
- *Done* - Finished today

Label *Todo* is used for items that are not yet on the schedule and in the beginning of the day you should archive all items in *Done*.

# Starting the bot locally

```
make run
# or
make docker
```
