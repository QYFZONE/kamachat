import { postFormData, postJSON } from "./http";

export function getUserInfo(uuid) {
  return postJSON("/user/getUserInfo", { uuid });
}

export function updateUserInfo(payload) {
  return postJSON("/user/updateUserInfo", payload);
}

export async function uploadAvatar(ownerId, file) {
  const formData = new FormData();
  formData.append("owner_id", ownerId);
  formData.append("avatar", file);

  const result = await postFormData("/message/uploadAvatar", formData);
  const extension = file.name.includes(".") ? file.name.slice(file.name.lastIndexOf(".")) : "";

  return {
    ...result,
    // The current backend stores avatars as avatar_ownerId.ext and serves them from /static/avatars.
    avatarPath: `/static/avatars/avatar_${ownerId}${extension}`,
  };
}

export function getFriendList(ownerId) {
  return postJSON("/contact/getUserList", { owner_id: ownerId });
}

export function getContactInfo(contactId) {
  return postJSON("/contact/getContactInfo", { contact_id: contactId });
}

export function applyContact(payload) {
  return postJSON("/contact/applyContact", payload);
}

export function getNewContactList(ownerId) {
  return postJSON("/contact/getNewContactList", { owner_id: ownerId });
}

export function passContactApply(ownerId, contactId) {
  return postJSON("/contact/passContactApply", { owner_id: ownerId, contact_id: contactId });
}

export function refuseContactApply(ownerId, contactId) {
  return postJSON("/contact/refuseContactApply", { owner_id: ownerId, contact_id: contactId });
}

export function blackApply(ownerId, contactId) {
  return postJSON("/contact/blackApply", { owner_id: ownerId, contact_id: contactId });
}

export function deleteContact(ownerId, contactId) {
  return postJSON("/contact/deleteContact", { owner_id: ownerId, contact_id: contactId });
}

export function blackContact(ownerId, contactId) {
  return postJSON("/contact/blackContact", { owner_id: ownerId, contact_id: contactId });
}

export function cancelBlackContact(ownerId, contactId) {
  return postJSON("/contact/cancelBlackContact", { owner_id: ownerId, contact_id: contactId });
}

export function getJoinedGroups(ownerId) {
  return postJSON("/contact/loadMyJoinedGroup", { owner_id: ownerId });
}

export function getCreatedGroups(ownerId) {
  return postJSON("/group/loadMyGroup", { owner_id: ownerId });
}

export function createGroup(payload) {
  return postJSON("/group/createGroup", payload);
}

export function getGroupInfo(groupId) {
  return postJSON("/group/getGroupInfo", { group_id: groupId });
}

export function getGroupMemberList(groupId) {
  return postJSON("/group/getGroupMemberList", { group_id: groupId });
}

export function updateGroupInfo(payload) {
  return postJSON("/group/updateGroupInfo", payload);
}

export function leaveGroup(userId, groupId) {
  return postJSON("/group/leaveGroup", { user_id: userId, group_id: groupId });
}

export function dismissGroup(ownerId, groupId) {
  return postJSON("/group/dismissGroup", { owner_id: ownerId, group_id: groupId });
}

export function removeGroupMembers(payload) {
  return postJSON("/group/removeGroupMembers", payload);
}

export function getUserSessions(ownerId) {
  return postJSON("/session/getUserSessionList", { owner_id: ownerId });
}

export function getGroupSessions(ownerId) {
  return postJSON("/session/getGroupSessionList", { owner_id: ownerId });
}

export function deleteSession(ownerId, sessionId) {
  return postJSON("/session/deleteSession", { owner_id: ownerId, session_id: sessionId });
}

export function checkOpenSessionAllowed(sendId, receiveId) {
  return postJSON("/session/checkOpenSessionAllowed", { send_id: sendId, receive_id: receiveId });
}

export function openSession(sendId, receiveId) {
  return postJSON("/session/openSession", { send_id: sendId, receive_id: receiveId });
}

export function getMessageList(userOneId, userTwoId) {
  return postJSON("/message/getMessageList", { user_one_id: userOneId, user_two_id: userTwoId });
}

export function getGroupMessageList(groupId) {
  return postJSON("/message/getGroupMessageList", { group_id: groupId });
}

export function uploadMessageFile(file) {
  const formData = new FormData();
  formData.append("file", file);
  return postFormData("/message/uploadFile", formData);
}

export function wsLogout(ownerId) {
  return postJSON("/wsLogout", { owner_id: ownerId });
}
