<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  ChatDotRound,
  Delete,
  Link,
  Message,
  Paperclip,
  Refresh,
  Search,
  SwitchButton,
  User,
  UserFilled,
} from "@element-plus/icons-vue";

import { buildWebSocketUrl, resolveAssetUrl } from "../api/http";
import {
  deleteSession,
  getGroupMessageList,
  getGroupSessions,
  getMessageList,
  getUserSessions,
  openSession,
  uploadMessageFile,
  wsLogout,
} from "../api/user";
import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser } from "../utils/storage";

const MESSAGE_TYPE_TEXT = 0;
const MESSAGE_TYPE_FILE = 2;

const router = useRouter();
const route = useRoute();
const currentUser = ref(getStoredUser() || {});
const loadingSessions = ref(false);
const loadingMessages = ref(false);
const sending = ref(false);
const uploading = ref(false);
const actionLoading = ref("");
const keyword = ref("");
const activeSessionType = ref("user");
const selectedSessionId = ref("");
const userSessions = ref([]);
const groupSessions = ref([]);
const messages = ref([]);
const draft = ref("");
const socket = ref(null);
const socketStatus = ref("disconnected");
const messageListRef = ref(null);
const fileInputRef = ref(null);

const activeSessions = computed(() => (activeSessionType.value === "group" ? groupSessions.value : userSessions.value));
const allSessions = computed(() => [...userSessions.value, ...groupSessions.value]);
const selectedSession = computed(() => allSessions.value.find((item) => item.session_id === selectedSessionId.value) || null);
const canSend = computed(() => Boolean(selectedSession.value?.target_id) && draft.value.trim().length > 0 && !sending.value);

const filteredSessions = computed(() => {
  const text = keyword.value.trim().toLowerCase();
  if (!text) {
    return activeSessions.value;
  }

  return activeSessions.value.filter((session) => {
    return [session.name, session.target_id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(text));
  });
});

const socketStatusText = computed(() => {
  if (socketStatus.value === "connected") {
    return "在线";
  }

  if (socketStatus.value === "connecting") {
    return "正在连接";
  }

  return "离线";
});

const socketStatusClass = computed(() => ({
  "is-online": socketStatus.value === "connected",
  "is-connecting": socketStatus.value === "connecting",
  "is-offline": socketStatus.value !== "connected" && socketStatus.value !== "connecting",
}));

function getInitials(name) {
  return String(name || "K").trim().slice(0, 1).toUpperCase();
}

function getAvatarUrl(avatar) {
  return resolveAssetUrl(avatar || "");
}

function formatMessageTime(value) {
  const text = String(value || "").trim();
  if (!text) {
    return "";
  }

  const date = new Date(text.replace(/-/g, "/"));
  if (Number.isNaN(date.getTime())) {
    return text;
  }

  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatFileSize(file) {
  const size = Number(file?.size || file || 0);
  if (!size) {
    return "0B";
  }

  if (size < 1024) {
    return `${size}B`;
  }

  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)}KB`;
  }

  return `${(size / 1024 / 1024).toFixed(1)}MB`;
}

function messageKey(message) {
  return [
    message.send_id,
    message.receive_id,
    message.type,
    message.content,
    message.url,
    message.created_at,
    message.file_name,
  ].join("|");
}

function normalizeSessions(data, type) {
  const list = Array.isArray(data) ? data : [];

  return list.map((session) => {
    if (type === "group") {
      return {
        type,
        session_id: session.session_id,
        target_id: session.group_id,
        name: session.group_name || "未命名群聊",
        avatar: session.avatar || "",
      };
    }

    return {
      type,
      session_id: session.session_id,
      target_id: session.user_id,
      name: session.user_name || "未命名好友",
      avatar: session.avatar || "",
    };
  });
}

function normalizeMessages(data) {
  const list = Array.isArray(data) ? data : [];

  return list
    .filter((message) => message?.send_id || message?.content || message?.url || message?.file_name)
    .map((message, index) => ({
      id: `${messageKey(message)}-${index}`,
      send_id: message.send_id || "",
      send_name: message.send_name || "",
      send_avatar: message.send_avatar || "",
      receive_id: message.receive_id || "",
      content: message.content || "",
      url: message.url || "",
      type: Number(message.type || 0),
      file_type: message.file_type || "",
      file_name: message.file_name || "",
      file_size: message.file_size || "",
      created_at: message.created_at || "",
    }));
}

function isOwnMessage(message) {
  return message.send_id === currentUser.value?.uuid;
}

function isMessageForSelectedSession(message) {
  const session = selectedSession.value;
  const userId = currentUser.value?.uuid;

  if (!session || !userId) {
    return false;
  }

  if (session.type === "group") {
    return message.receive_id === session.target_id;
  }

  return (
    (message.send_id === userId && message.receive_id === session.target_id) ||
    (message.send_id === session.target_id && message.receive_id === userId)
  );
}

async function ensureUser() {
  const user = getStoredUser();
  if (!user?.uuid) {
    clearStoredUser();
    await router.push("/auth");
    return null;
  }

  currentUser.value = user;
  return user;
}

function scrollMessagesToBottom() {
  nextTick(() => {
    const el = messageListRef.value;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  });
}

function upsertQuerySession() {
  const queryType = route.query.type === "group" ? "group" : route.query.type === "user" ? "user" : "";
  const targetId = String(route.query.id || "");
  const sessionId = String(route.query.session_id || "");

  if (!queryType || !targetId) {
    return false;
  }

  activeSessionType.value = queryType;

  const listRef = queryType === "group" ? groupSessions : userSessions;
  const existing = listRef.value.find((session) => session.session_id === sessionId || session.target_id === targetId);

  if (existing) {
    selectedSessionId.value = existing.session_id;
    return true;
  }

  const localSession = {
    type: queryType,
    session_id: sessionId || `pending-${targetId}`,
    target_id: targetId,
    name: String(route.query.name || (queryType === "group" ? "群聊" : "好友")),
    avatar: String(route.query.avatar || ""),
  };

  listRef.value = [localSession, ...listRef.value];
  selectedSessionId.value = localSession.session_id;
  return true;
}

function selectFirstAvailableSession() {
  if (selectedSession.value) {
    return;
  }

  selectedSessionId.value = activeSessions.value[0]?.session_id || allSessions.value[0]?.session_id || "";
  const nextSession = selectedSession.value;
  if (nextSession) {
    activeSessionType.value = nextSession.type;
  }
}

async function loadSessions() {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  loadingSessions.value = true;

  try {
    const [userResult, groupResult] = await Promise.allSettled([
      getUserSessions(user.uuid),
      getGroupSessions(user.uuid),
    ]);

    userSessions.value = userResult.status === "fulfilled" ? normalizeSessions(userResult.value.data, "user") : [];
    groupSessions.value = groupResult.status === "fulfilled" ? normalizeSessions(groupResult.value.data, "group") : [];

    upsertQuerySession();
    selectFirstAvailableSession();

    if (selectedSession.value) {
      await loadMessages();
    } else {
      messages.value = [];
    }
  } catch (error) {
    ElMessage.error(error?.message || "聊天列表加载失败");
  } finally {
    loadingSessions.value = false;
  }
}

async function loadMessages() {
  const user = await ensureUser();
  const session = selectedSession.value;

  if (!user || !session?.target_id) {
    messages.value = [];
    return;
  }

  loadingMessages.value = true;

  try {
    const result =
      session.type === "group"
        ? await getGroupMessageList(session.target_id)
        : await getMessageList(user.uuid, session.target_id);

    messages.value = normalizeMessages(result.data);
    scrollMessagesToBottom();
  } catch (error) {
    messages.value = [];
    ElMessage.warning(error?.message || "消息记录加载失败");
  } finally {
    loadingMessages.value = false;
  }
}

function handleSocketMessage(event) {
  let payload;

  try {
    payload = JSON.parse(event.data);
  } catch (_error) {
    return;
  }

  const nextMessage = normalizeMessages([payload])[0];
  if (!nextMessage) {
    return;
  }

  if (!isMessageForSelectedSession(nextMessage)) {
    return;
  }

  const nextKey = messageKey(nextMessage);
  if (messages.value.some((message) => messageKey(message) === nextKey)) {
    return;
  }

  messages.value = [...messages.value, nextMessage];
  scrollMessagesToBottom();
}

function connectSocket() {
  const user = currentUser.value;
  if (!user?.uuid || socketStatus.value === "connected" || socketStatus.value === "connecting") {
    return;
  }

  socketStatus.value = "connecting";
  const ws = new WebSocket(buildWebSocketUrl(`/wss?client_id=${encodeURIComponent(user.uuid)}`));
  socket.value = ws;

  ws.onopen = () => {
    socketStatus.value = "connected";
  };

  ws.onmessage = handleSocketMessage;

  ws.onerror = () => {
    socketStatus.value = "disconnected";
  };

  ws.onclose = () => {
    if (socket.value === ws) {
      socketStatus.value = "disconnected";
      socket.value = null;
    }
  };
}

async function ensureSocketReady() {
  connectSocket();

  if (socket.value?.readyState === WebSocket.OPEN) {
    return true;
  }

  await new Promise((resolve) => {
    let attempts = 0;
    const timer = window.setInterval(() => {
      attempts += 1;
      if (socket.value?.readyState === WebSocket.OPEN || attempts > 20) {
        window.clearInterval(timer);
        resolve();
      }
    }, 100);
  });

  return socket.value?.readyState === WebSocket.OPEN;
}

async function ensureRealSession(session) {
  const user = await ensureUser();
  if (!user || !session) {
    return null;
  }

  if (!String(session.session_id).startsWith("pending-")) {
    return session;
  }

  const result = await openSession(user.uuid, session.target_id);
  const realSessionId = result.data;

  const patchSession = (item) =>
    item.session_id === session.session_id
      ? {
          ...item,
          session_id: realSessionId,
        }
      : item;

  if (session.type === "group") {
    groupSessions.value = groupSessions.value.map(patchSession);
  } else {
    userSessions.value = userSessions.value.map(patchSession);
  }

  selectedSessionId.value = realSessionId;
  return selectedSession.value;
}

async function sendPayload(payload) {
  const ready = await ensureSocketReady();
  if (!ready) {
    throw new Error("消息连接未建立，请稍后重试");
  }

  socket.value.send(JSON.stringify(payload));
}

async function sendTextMessage() {
  const user = await ensureUser();
  const session = await ensureRealSession(selectedSession.value);
  const content = draft.value.trim();

  if (!user || !session || !content) {
    return;
  }

  sending.value = true;

  try {
    await sendPayload({
      session_id: session.session_id,
      type: MESSAGE_TYPE_TEXT,
      content,
      url: "",
      send_id: user.uuid,
      send_name: user.nickname || "我",
      send_avatar: user.avatar || "",
      receive_id: session.target_id,
      file_size: "0B",
      file_type: "",
      file_name: "",
      av_data: "",
    });
    draft.value = "";
  } catch (error) {
    ElMessage.error(error?.message || "消息发送失败");
  } finally {
    sending.value = false;
  }
}

function triggerFilePicker() {
  fileInputRef.value?.click();
}

async function handleFileChange(event) {
  const file = event.target.files?.[0];
  event.target.value = "";

  if (!file) {
    return;
  }

  const user = await ensureUser();
  const session = await ensureRealSession(selectedSession.value);
  if (!user || !session) {
    return;
  }

  uploading.value = true;

  try {
    const result = await uploadMessageFile(file);
    const url = result.data;
    if (!url) {
      throw new Error("文件上传失败");
    }

    await sendPayload({
      session_id: session.session_id,
      type: MESSAGE_TYPE_FILE,
      content: file.name,
      url,
      send_id: user.uuid,
      send_name: user.nickname || "我",
      send_avatar: user.avatar || "",
      receive_id: session.target_id,
      file_size: formatFileSize(file),
      file_type: file.type || "file",
      file_name: file.name,
      av_data: "",
    });
  } catch (error) {
    ElMessage.error(error?.message || "文件发送失败");
  } finally {
    uploading.value = false;
  }
}

async function selectSession(session) {
  activeSessionType.value = session.type;
  selectedSessionId.value = session.session_id;
  await loadMessages();
}

async function removeSelectedSession() {
  const user = await ensureUser();
  const session = selectedSession.value;
  if (!user || !session) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定删除「${session.name}」这段聊天吗？`, "删除聊天", {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = "delete-session";

  try {
    const result = await deleteSession(user.uuid, session.session_id);
    ElMessage.success(result.message || "聊天已删除");
    selectedSessionId.value = "";
    await loadSessions();
  } catch (error) {
    ElMessage.error(error?.message || "删除聊天失败");
  } finally {
    actionLoading.value = "";
  }
}

async function refreshPage() {
  await loadSessions();
}

async function goHome() {
  await router.push("/");
}

async function goFriends() {
  await router.push({ name: "friends" });
}

async function goGroups() {
  await router.push({ name: "groups" });
}

async function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  await router.push("/auth");
}

function syncSelectedSessionWithVisibleList() {
  const visibleSessions = filteredSessions.value;

  if (!visibleSessions.length) {
    if (keyword.value.trim() || !activeSessions.value.length) {
      selectedSessionId.value = "";
      messages.value = [];
    }
    return;
  }

  if (visibleSessions.some((session) => session.session_id === selectedSessionId.value)) {
    return;
  }

  selectedSessionId.value = visibleSessions[0].session_id;
  loadMessages();
}

watch(
  () => [activeSessionType.value, keyword.value.trim(), filteredSessions.value.map((session) => session.session_id).join("|")],
  () => {
    syncSelectedSessionWithVisibleList();
  },
);

watch(
  () => route.query,
  async () => {
    if (upsertQuerySession()) {
      await loadMessages();
    }
  },
);

onMounted(async () => {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  connectSocket();
  await loadSessions();
});

onBeforeUnmount(() => {
  const user = currentUser.value;
  if (socket.value) {
    socket.value.close();
  }
  if (user?.uuid) {
    wsLogout(user.uuid).catch(() => {});
  }
});
</script>

<template>
  <main class="messages-page">
    <section class="page-shell messages-page__shell">
      <header class="glass-card messages-navbar">
        <div class="messages-navbar__brand">
          <button type="button" class="messages-navbar__back" @click="goHome">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <span class="messages-navbar__logo">K</span>
          <div>
            <p class="messages-navbar__eyebrow">KamaChat</p>
            <h1 class="messages-navbar__title">我的消息</h1>
          </div>
        </div>

        <div class="messages-navbar__actions">
          <span class="messages-status" :class="socketStatusClass">
            <span></span>
            {{ socketStatusText }}
          </span>
          <el-button plain @click="refreshPage">
            <el-icon><Refresh /></el-icon>
            <span>刷新</span>
          </el-button>
          <el-button type="primary" plain @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </el-button>
        </div>
      </header>

      <section class="messages-layout">
        <aside class="glass-card messages-sidebar">
          <header class="messages-panel-heading">
            <div>
              <p class="messages-panel-heading__eyebrow">聊天</p>
              <h2 class="messages-panel-heading__title">聊天列表</h2>
            </div>
            <span class="messages-panel-heading__badge">{{ activeSessions.length }}</span>
          </header>

          <div class="messages-switch" :data-mode="activeSessionType">
            <span class="messages-switch__thumb"></span>
            <button
              type="button"
              class="messages-switch__button"
              :class="{ 'is-active': activeSessionType === 'user' }"
              @click="activeSessionType = 'user'"
            >
              单聊
            </button>
            <button
              type="button"
              class="messages-switch__button"
              :class="{ 'is-active': activeSessionType === 'group' }"
              @click="activeSessionType = 'group'"
            >
              群聊
            </button>
          </div>

          <el-input v-model="keyword" class="messages-search" placeholder="搜索当前聊天" clearable>
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <div v-loading="loadingSessions" class="session-list">
            <el-empty
              v-if="!filteredSessions.length && !loadingSessions"
              :description="keyword.trim() ? '没有找到聊天' : '暂无聊天'"
            >
              <div class="session-empty-actions">
                <el-button size="small" @click="goFriends">找好友</el-button>
                <el-button size="small" @click="goGroups">看群聊</el-button>
              </div>
            </el-empty>

            <button
              v-for="session in filteredSessions"
              :key="session.session_id"
              type="button"
              class="session-card"
              :class="{ 'is-active': selectedSessionId === session.session_id }"
              @click="selectSession(session)"
            >
              <el-avatar :size="36" :src="getAvatarUrl(session.avatar)" class="session-card__avatar">
                {{ getInitials(session.name) }}
              </el-avatar>
              <span class="session-card__body">
                <strong>{{ session.name }}</strong>
                <small>{{ session.type === "group" ? "群聊" : "好友" }}</small>
              </span>
              <span class="session-card__tag">{{ session.type === "group" ? "群" : "聊" }}</span>
            </button>
          </div>
        </aside>

        <section class="glass-card chat-panel">
          <template v-if="selectedSession">
            <header class="chat-header">
              <div class="chat-header__identity">
                <el-avatar :size="42" :src="getAvatarUrl(selectedSession.avatar)" class="chat-header__avatar">
                  {{ getInitials(selectedSession.name) }}
                </el-avatar>
                <div>
                  <p class="chat-header__eyebrow">{{ selectedSession.type === "group" ? "群聊" : "单聊" }}</p>
                  <h2>{{ selectedSession.name }}</h2>
                  <span>正在聊天</span>
                </div>
              </div>

              <div class="chat-header__actions">
                <el-button plain @click="loadMessages">
                  <el-icon><Refresh /></el-icon>
                </el-button>
                <el-button plain :loading="actionLoading === 'delete-session'" @click="removeSelectedSession">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </header>

            <div ref="messageListRef" v-loading="loadingMessages" class="message-list">
              <el-empty v-if="!messages.length && !loadingMessages" description="还没有消息" />

              <article
                v-for="messageItem in messages"
                :key="messageItem.id"
                class="message-item"
                :class="{ 'is-mine': isOwnMessage(messageItem) }"
              >
                <el-avatar :size="30" :src="getAvatarUrl(messageItem.send_avatar)" class="message-item__avatar">
                  {{ getInitials(messageItem.send_name) }}
                </el-avatar>

                <div class="message-item__content">
                  <div class="message-item__meta">
                    <strong>{{ isOwnMessage(messageItem) ? "我" : messageItem.send_name || "好友" }}</strong>
                    <span>{{ formatMessageTime(messageItem.created_at) }}</span>
                  </div>

                  <div class="message-bubble">
                    <template v-if="messageItem.type === MESSAGE_TYPE_FILE">
                      <a class="message-file" :href="getAvatarUrl(messageItem.url)" target="_blank" rel="noreferrer">
                        <span class="message-file__icon">
                          <el-icon><Link /></el-icon>
                        </span>
                        <span>
                          <strong>{{ messageItem.file_name || messageItem.content || "文件" }}</strong>
                          <small>{{ messageItem.file_size || "未知大小" }}</small>
                        </span>
                      </a>
                    </template>
                    <template v-else>
                      {{ messageItem.content || "消息内容为空" }}
                    </template>
                  </div>
                </div>
              </article>
            </div>

            <footer class="chat-composer">
              <input ref="fileInputRef" type="file" class="chat-composer__file" @change="handleFileChange" />
              <el-button plain :disabled="uploading" :loading="uploading" @click="triggerFilePicker">
                <el-icon><Paperclip /></el-icon>
              </el-button>
              <el-input
                v-model="draft"
                type="textarea"
                :rows="2"
                maxlength="500"
                resize="none"
                placeholder="输入消息"
                @keydown.enter.exact.prevent="sendTextMessage"
              />
              <el-button type="primary" :disabled="!canSend" :loading="sending" @click="sendTextMessage">
                <el-icon><Message /></el-icon>
                <span>发送</span>
              </el-button>
            </footer>
          </template>

          <div v-else class="chat-empty">
            <span class="chat-empty__icon">
              <el-icon><ChatDotRound /></el-icon>
            </span>
            <h2>选择一个聊天</h2>
            <p>从左侧选择好友或群聊。</p>
            <div>
              <el-button @click="goFriends">
                <el-icon><User /></el-icon>
                <span>我的好友</span>
              </el-button>
              <el-button type="primary" @click="goGroups">
                <el-icon><UserFilled /></el-icon>
                <span>我的群聊</span>
              </el-button>
            </div>
          </div>
        </section>
      </section>
    </section>
  </main>
</template>

<style scoped>
.messages-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 14% 12%, rgba(255, 255, 255, 0.86) 0, rgba(255, 255, 255, 0) 27%),
    radial-gradient(circle at 82% 14%, rgba(255, 214, 188, 0.42) 0, rgba(255, 214, 188, 0) 27%),
    linear-gradient(135deg, #fff8f2 0%, #ffe8d8 48%, #f7f0e8 100%);
}

.messages-page__shell {
  display: grid;
  gap: 14px;
  width: min(1120px, calc(100% - 32px));
}

.messages-navbar,
.messages-navbar__brand,
.messages-navbar__actions {
  display: flex;
  align-items: center;
}

.messages-navbar {
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.messages-navbar__brand,
.messages-navbar__actions {
  gap: 14px;
}

.messages-navbar__back {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(220, 177, 150, 0.46);
  border-radius: 14px;
  background: rgba(255, 249, 245, 0.92);
  color: #9d4d2d;
  cursor: pointer;
}

.messages-navbar__logo {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: linear-gradient(135deg, #f7bf96 0%, #ec9b66 100%);
  color: #ffffff;
  font-size: 18px;
  font-weight: 800;
  box-shadow: 0 10px 18px rgba(188, 90, 52, 0.18);
}

.messages-navbar__eyebrow,
.messages-navbar__title {
  margin: 0;
}

.messages-navbar__eyebrow {
  color: var(--kc-muted);
  font-size: 12px;
}

.messages-navbar__title {
  font-size: 18px;
  font-weight: 800;
}

.messages-status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 13px;
  border-radius: 999px;
  background: rgba(244, 232, 220, 0.94);
  color: #7d6355;
  font-size: 13px;
  font-weight: 800;
}

.messages-status span {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: currentColor;
}

.messages-status.is-online {
  background: rgba(222, 247, 233, 0.88);
  color: #1f7656;
}

.messages-status.is-connecting {
  background: #fff0e2;
  color: #9d4d2d;
}

.messages-panel-heading__eyebrow,
.chat-header__eyebrow {
  display: none;
  margin: 0;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.messages-layout {
  display: grid;
  grid-template-columns: minmax(230px, 280px) minmax(0, 1fr);
  gap: 12px;
  height: min(680px, calc(100vh - 124px));
  min-height: 520px;
}

.messages-sidebar,
.chat-panel {
  padding: 14px;
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(18px);
}

.messages-sidebar {
  align-self: start;
  min-height: 0;
  max-height: 100%;
  overflow: visible;
}

.messages-panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.messages-panel-heading__title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}

.messages-panel-heading__badge {
  min-width: 32px;
  padding: 7px 10px;
  border-radius: 999px;
  background: #fff0e2;
  color: #9d4d2d;
  font-size: 12px;
  font-weight: 800;
  text-align: center;
}

.messages-switch {
  position: relative;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 8px;
  padding: 3px;
  border: 1px solid rgba(213, 166, 140, 0.36);
  border-radius: 999px;
  background: rgba(253, 245, 238, 0.95);
}

.messages-switch__thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: calc(50% - 3px);
  height: calc(100% - 6px);
  border-radius: 999px;
  background: linear-gradient(135deg, #ffc89c 0%, #f1aa75 100%);
  box-shadow: 0 10px 22px rgba(185, 107, 63, 0.16);
  transition: transform 0.25s ease;
}

.messages-switch[data-mode="group"] .messages-switch__thumb {
  transform: translateX(calc(100% + 3px));
}

.messages-switch__button {
  position: relative;
  z-index: 1;
  min-height: 32px;
  border: 0;
  background: transparent;
  color: #836355;
  font: inherit;
  font-weight: 800;
  cursor: pointer;
}

.messages-switch__button.is-active {
  color: #5e3421;
}

.messages-search {
  position: relative;
  z-index: 1;
  margin-bottom: 12px;
}

.messages-search :deep(.el-input__wrapper),
.chat-composer :deep(.el-textarea__inner) {
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
}

.messages-search :deep(.el-input__wrapper) {
  min-height: 34px;
}

.session-list {
  position: relative;
  z-index: 0;
  display: grid;
  gap: 6px;
  min-height: 0;
  max-height: min(420px, calc(100vh - 292px));
  overflow-y: auto;
  padding: 2px 4px 2px 0;
  scrollbar-gutter: stable;
}

.session-list :deep(.el-empty) {
  padding: 28px 0;
}

.session-list :deep(.el-empty__image) {
  width: 132px;
}

.session-empty-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 10px;
}

.session-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 9px;
  align-items: center;
  width: 100%;
  min-height: 52px;
  padding: 8px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 14px;
  background: rgba(255, 250, 246, 0.72);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.session-card:hover,
.session-card.is-active {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.34);
  box-shadow: 0 12px 22px rgba(145, 87, 58, 0.08);
}

.session-card.is-active {
  background: rgba(255, 239, 227, 0.88);
}

.session-card__avatar,
.chat-header__avatar,
.message-item__avatar {
  background: linear-gradient(135deg, #fff3e8 0%, #f7b17e 100%);
  color: #8f4829;
  font-weight: 800;
}

.session-card__body {
  min-width: 0;
}

.session-card__body strong,
.session-card__body small {
  display: block;
}

.session-card__body strong {
  overflow: hidden;
  color: #2f211a;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-card__body small {
  margin-top: 2px;
  color: var(--kc-muted);
  font-size: 12px;
}

.session-card__tag {
  display: none;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 999px;
  background: rgba(222, 247, 233, 0.88);
  color: #1f7656;
  font-size: 12px;
  font-weight: 800;
}

.chat-panel {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 14px;
  background: linear-gradient(145deg, #fff8f1 0%, #ffd9c0 100%);
}

.chat-header__identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.chat-header__identity div {
  min-width: 0;
}

.chat-header__identity h2,
.chat-header__identity span {
  margin: 0;
}

.chat-header__identity h2 {
  overflow: hidden;
  color: #2f211a;
  font-size: 16px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-header__identity span {
  display: none;
  margin-top: 0;
  overflow-wrap: anywhere;
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.chat-header__actions {
  display: flex;
  gap: 8px;
}

.chat-header__actions :deep(.el-button) {
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: 12px;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  margin: 10px 0;
  padding: 12px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 16px;
  background: rgba(255, 250, 246, 0.62);
}

.message-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 10px;
  align-items: flex-start;
  max-width: min(74%, 620px);
}

.message-item.is-mine {
  grid-template-columns: minmax(0, 1fr) auto;
  align-self: flex-end;
}

.message-item.is-mine .message-item__avatar {
  grid-column: 2;
  grid-row: 1;
}

.message-item.is-mine .message-item__content {
  grid-column: 1;
  grid-row: 1;
  align-items: flex-end;
}

.message-item__content {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.message-item__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--kc-muted);
  font-size: 12px;
}

.message-item.is-mine .message-item__meta {
  justify-content: flex-end;
}

.message-item__meta strong {
  color: #5e3421;
}

.message-bubble {
  width: fit-content;
  max-width: 100%;
  padding: 10px 12px;
  border: 1px solid rgba(232, 224, 214, 0.9);
  border-radius: 18px;
  background: #ffffff;
  color: #4c3428;
  line-height: 1.55;
  overflow-wrap: anywhere;
  box-shadow: 0 12px 24px rgba(145, 87, 58, 0.06);
}

.message-item.is-mine .message-bubble {
  border-color: rgba(230, 154, 103, 0.4);
  background: linear-gradient(135deg, #ffc89c 0%, #f1aa75 100%);
  color: #4a2513;
}

.message-file {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  color: inherit;
}

.message-file__icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
}

.message-file strong,
.message-file small {
  display: block;
}

.message-file strong {
  overflow-wrap: anywhere;
}

.message-file small {
  margin-top: 4px;
  color: inherit;
  opacity: 0.72;
}

.chat-composer {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 68px;
  gap: 7px;
  align-items: end;
  flex: 0 0 auto;
  padding-top: 6px;
  border-top: 1px solid rgba(232, 224, 214, 0.82);
  background: rgba(255, 255, 255, 0.76);
}

.chat-composer__file {
  display: none;
}

.chat-composer :deep(.el-button) {
  min-height: 36px;
  height: 36px;
  border-radius: 12px;
}

.chat-composer :deep(.el-textarea__inner) {
  min-height: 40px;
  max-height: 72px;
  padding-top: 8px;
  padding-bottom: 8px;
}

.chat-composer :deep(.el-button--primary) {
  min-width: 68px;
}

.chat-empty {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  min-height: 100%;
  text-align: center;
}

.chat-empty__icon {
  display: grid;
  place-items: center;
  width: 74px;
  height: 74px;
  border-radius: 24px;
  background: #fff0e2;
  color: #9d4d2d;
  font-size: 28px;
}

.chat-empty h2,
.chat-empty p {
  margin: 0;
}

.chat-empty p {
  color: var(--kc-muted);
}

.chat-empty div {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  margin-top: 8px;
}

@media (max-width: 1080px) {
  .messages-layout {
    grid-template-columns: 1fr;
  }

  .messages-layout {
    height: auto;
    min-height: 0;
  }

  .messages-sidebar {
    min-height: 188px;
  }

  .chat-panel {
    height: min(68vh, 620px);
    min-height: 460px;
  }

  .message-list {
    min-height: 0;
  }
}

@media (max-width: 760px) {
  .messages-navbar,
  .messages-navbar__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .messages-sidebar {
    min-height: 0;
  }

  .session-list {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(210px, 76%);
    grid-template-columns: none;
    max-height: none;
    overflow-x: auto;
    overflow-y: hidden;
    padding: 2px 0 8px;
    scroll-snap-type: x mandatory;
    scrollbar-width: thin;
  }

  .session-card {
    scroll-snap-align: start;
  }

  .chat-header {
    align-items: center;
    flex-direction: row;
  }

  .chat-header__actions {
    flex-shrink: 0;
  }

  .message-item,
  .message-item.is-mine {
    max-width: 92%;
  }

  .chat-composer {
    grid-template-columns: 36px minmax(0, 1fr) 64px;
    gap: 7px;
  }
}

@media (max-width: 560px) {
  .messages-page__shell {
    width: min(100% - 20px, 1120px);
  }

  .messages-navbar,
  .messages-sidebar,
  .chat-panel {
    padding: 12px;
  }

  .chat-panel {
    height: 420px;
    min-height: 420px;
  }

  .session-list :deep(.el-empty) {
    padding: 18px 0;
  }

  .session-list :deep(.el-empty__image) {
    width: 96px;
  }

  .session-card {
    grid-template-columns: auto minmax(0, 1fr);
    min-height: 48px;
  }

  .session-card__tag {
    grid-column: 2;
    justify-self: start;
  }

  .message-list {
    margin: 10px 0;
    padding: 10px;
  }

  .message-item,
  .message-item.is-mine {
    max-width: 100%;
  }
}
</style>
