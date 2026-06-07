I want to create a session manager for tmux which manages open ai sessions like tmux session manager does: https://github.com/omerxx/tmux-sessionx

Here's the flow I am thinking about: 
1. When you enter a shortcut, a floating window pops up like tmux-session which shows open ai sessions. You can navigate through the sessions. These live as new tmux session scoped to the repo in which an ai session should live. If no repo is specificed, it would run in a generic "temp" repo. 
2. In this floating window, you could create new AI sessions. When creating a session, the user should be prompted with a fzf searchable list of repos (perhaps users can select a "root" repo, like I use a "src" or "code" folder to store all of my git repos). The user can then select the correct repo they want the session in. As a stretch goal, ideally, this would integrate with git worktree or worktrunk (wt) cli to create brand new WTs so AIs are fighting over eachother in the same repo. 
3. Users can also delete sessions from this window when they are done with them. Deleting a session will:
    - delete the sessoin
    - delete any open work trees
    - delete the running agent

Does this align? Lets try to scope out this project. What tools will be needed? What language should be used if this were to work with tmux? (I am most familiar with go). Once we get a good idea and spec defined, then we can work on breaking these things out as tasks to have sonnet 4.6 agents work on to save on tokens.
