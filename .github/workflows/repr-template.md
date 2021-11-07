### checkout this branch
```
mkdir -p $HOME/1o-fyi/lofi-node/{branch_name} && cd $HOME/1o-fyi/lofi-node/{branch_name}
git init -b {branch_name}
git remote add origin https://github.com/1o-fyi/lofi-node.git
git fetch origin --tags && git fetch origin pull/{pull_id}/head:{branch_name}
git checkout {branch_name}
``` 

