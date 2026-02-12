let groups = $state([]);

export function getGroups() { return groups; }

export function setGroups(list) {
  groups = list;
}

export function addOrUpdateGroup(group) {
  const idx = groups.findIndex(g => g.id === group.id);
  if (idx >= 0) {
    groups[idx] = group;
  } else {
    groups = [...groups, group];
  }
}

export function removeGroup(groupId) {
  groups = groups.filter(g => g.id !== groupId);
}
