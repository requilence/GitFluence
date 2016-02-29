GitFluence
============================

Github repo visualization using **git-blame** in a unusual way. GitFluence clone repo, parse git-blame output of all files in the repo, determine the type of each file(code, docs, test or resource) and the type of each line(comment or code). This data used to create the influence map of each contributor by the actual repo's codebase. This looks convenient since contributors tab on GitHub doesn't show the actual big picture of project's codebase. 

Selection of line types(code, docs, test or resource) and time(last month, 3 months, 6 months or 1year) is available.

![](https://1571124647.rsc.cdn77.org/gitfluence_sample.png  "Example: google/go-github repo")

#### Koding Hackathon submission
After hackathon judging will end I planning to finish rating calculation based on this algorithm 
