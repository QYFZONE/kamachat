<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  ChatDotRound,
  Check,
  CirclePlus,
  Close,
  Delete,
  House,
  Iphone,
  Lock,
  Message,
  Refresh,
  Search,
  SwitchButton,
  Unlock,
  User,
  UserFilled,
  Warning,
} from "@element-plus/icons-vue";

import { resolveAssetUrl } from "../api/http";
import {
  applyContact,
  blackApply,
  blackContact,
  cancelBlackContact,
  checkOpenSessionAllowed,
  deleteContact,
  getContactInfo,
  getFriendList,
  getNewContactList,
  openSession,
  passContactApply,
  refuseContactApply,
} from "../api/user";
import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser } from "../utils/storage";

const router = useRouter();
const currentUser = ref(getStoredUser() || {});
const loading = ref(false);
const detailLoading = ref(false);
const applying = ref(false);
const applicationLoading = ref(false);
const actionLoading = ref("");
const keyword = ref("");
const selectedFriendId = ref("");
const activeSection = ref("friends");
const addFriendDialogVisible = ref(false);
const friends = ref([]);
const applications = ref([]);
const selectedDetail = ref(null);
const contactStatusOverrides = ref({});
const pendingFriendAdds = ref({});
const pendingFriendRemovals = ref({});

const addForm = reactive({
  contactId: "",
  message: "你好，我想加你为好友。",
});

const selectedFriend = computed(() => {
  return friends.value.find((friend) => friend.user_id === selectedFriendId.value) || null;
});

const selectedFriendStatus = computed(() => getFriendStatus(selectedFriend.value));
const selectedFriendBlockedByMe = computed(() => Number(selectedFriendStatus.value) === 2);

const filteredFriends = computed(() => {
  const text = keyword.value.trim().toLowerCase();
  if (!text) {
    return friends.value;
  }

  return friends.value.filter((friend) => {
    return [friend.user_name, friend.user_id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(text));
  });
});

const friendStats = computed(() => [
  {
    label: "好友总数",
    value: friends.value.length,
    icon: UserFilled,
    tone: "peach",
  },
  {
    label: "待处理申请",
    value: applications.value.length,
    icon: CirclePlus,
    tone: "rose",
  },
  {
    label: "在线",
    value: "在线",
    icon: ChatDotRound,
    tone: "mint",
  },
]);

const detailCards = computed(() => {
  const detail = selectedDetail.value || {};

  return [
    {
      label: "手机号",
      value: detail.contact_phone || "未填写",
      icon: Iphone,
    },
    {
      label: "邮箱",
      value: detail.contact_email || "未填写",
      icon: Message,
    },
    {
      label: "性别",
      value: formatGender(detail.contact_gender),
      icon: User,
    },
  ];
});

function getInitials(name) {
  return String(name || "K").trim().slice(0, 1).toUpperCase();
}

function getAvatarUrl(avatar) {
  return resolveAssetUrl(avatar || "");
}

function formatGender(value) {
  if (value === 0) {
    return "男";
  }

  if (value === 1) {
    return "女";
  }

  return "未填写";
}

function formatBirthday(value) {
  const text = String(value || "").trim();
  if (!text) {
    return "未填写";
  }

  if (/^\d{8}$/.test(text)) {
    return `${text.slice(0, 4)}-${text.slice(4, 6)}-${text.slice(6, 8)}`;
  }

  return text;
}

function contactStatusText(status) {
  const value = Number(status);

  if (value === 2) {
    return "已拉黑";
  }

  if (value === 1) {
    return "被拉黑";
  }

  return "在线";
}

function getFriendStatus(friend) {
  if (!friend?.user_id) {
    return 0;
  }

  return contactStatusOverrides.value[friend.user_id] ?? friend.status ?? 0;
}

function contactStatusClass(status) {
  const value = Number(status);

  if (value === 2) {
    return "is-blocked";
  }

  if (value === 1) {
    return "is-blocked-by";
  }

  return "is-online";
}

function normalizeFriendList(data) {
  if (!Array.isArray(data)) {
    return [];
  }

  return data.map((friend) => ({
    ...friend,
    status: contactStatusOverrides.value[friend.user_id] ?? friend.status ?? 0,
  }));
}

function patchFriendList(list) {
  const serverIds = new Set(list.map((friend) => friend.user_id));
  const nextAdds = { ...pendingFriendAdds.value };
  const nextRemovals = { ...pendingFriendRemovals.value };

  for (const userId of serverIds) {
    delete nextAdds[userId];
  }

  for (const userId of Object.keys(nextRemovals)) {
    if (!serverIds.has(userId)) {
      delete nextRemovals[userId];
    }
  }

  pendingFriendAdds.value = nextAdds;
  pendingFriendRemovals.value = nextRemovals;

  const visibleList = list.filter((friend) => !nextRemovals[friend.user_id]);
  const visibleIds = new Set(visibleList.map((friend) => friend.user_id));
  const localAdds = Object.values(nextAdds).filter((friend) => !visibleIds.has(friend.user_id));

  return [...localAdds, ...visibleList];
}

function syncFriendsFromServer(data) {
  friends.value = patchFriendList(normalizeFriendList(data));
}

function removeApplicationLocally(contactId) {
  applications.value = applications.value.filter((application) => application.contact_id !== contactId);
}

function rememberLocalFriendAdd(application) {
  const friend = {
    user_id: application.contact_id,
    user_name: application.contact_name,
    avatar: application.contact_avatar,
    status: 0,
  };

  pendingFriendAdds.value = {
    ...pendingFriendAdds.value,
    [friend.user_id]: friend,
  };

  const { [friend.user_id]: _removed, ...remainingRemovals } = pendingFriendRemovals.value;
  pendingFriendRemovals.value = remainingRemovals;
  friends.value = patchFriendList(friends.value);
}

function rememberLocalFriendRemoval(friendId) {
  pendingFriendRemovals.value = {
    ...pendingFriendRemovals.value,
    [friendId]: true,
  };

  const { [friendId]: _added, ...remainingAdds } = pendingFriendAdds.value;
  pendingFriendAdds.value = remainingAdds;
  friends.value = patchFriendList(friends.value);
}

function selectAvailableFriend() {
  const previousFriendId = selectedFriendId.value;

  if (!selectedFriendId.value || !friends.value.some((friend) => friend.user_id === selectedFriendId.value)) {
    selectedFriendId.value = friends.value[0]?.user_id || "";
  }

  if (!selectedFriendId.value || selectedFriendId.value !== previousFriendId) {
    selectedDetail.value = null;
  }
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

async function loadFriends() {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  loading.value = true;

  try {
    const result = await getFriendList(user.uuid);
    syncFriendsFromServer(result.data);
    selectAvailableFriend();

    if (selectedFriendId.value) {
      await loadFriendDetail(selectedFriendId.value);
    } else {
      selectedDetail.value = null;
    }
  } catch (error) {
    friends.value = [];
    selectedFriendId.value = "";
    selectedDetail.value = null;
    ElMessage.error(error?.message || "好友列表加载失败");
  } finally {
    loading.value = false;
  }
}

async function loadApplications() {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  applicationLoading.value = true;

  try {
    const result = await getNewContactList(user.uuid);
    applications.value = Array.isArray(result.data) ? result.data : [];
  } catch (error) {
    applications.value = [];
    ElMessage.warning(error?.message || "好友申请加载失败");
  } finally {
    applicationLoading.value = false;
  }
}

async function refreshAll() {
  await Promise.all([loadFriends(), loadApplications()]);
}

async function loadFriendDetail(friendId) {
  if (!friendId) {
    selectedDetail.value = null;
    return;
  }

  selectedFriendId.value = friendId;
  detailLoading.value = true;

  try {
    const result = await getContactInfo(friendId);
    selectedDetail.value = result.data || null;
  } catch (error) {
    selectedDetail.value = null;
    ElMessage.warning(error?.message || "好友详情加载失败");
  } finally {
    detailLoading.value = false;
  }
}

async function submitAddFriend() {
  const user = await ensureUser();
  const contactId = addForm.contactId.trim();

  if (!user || applying.value) {
    return;
  }

  if (!/^U[A-Za-z0-9]+$/.test(contactId)) {
    ElMessage.error("请输入正确的好友账号");
    return;
  }

  applying.value = true;

  try {
    const result = await applyContact({
      owner_id: user.uuid,
      contact_id: contactId,
      message: addForm.message.trim(),
    });
    ElMessage.success(result.message || "好友申请已发送");
    addForm.contactId = "";
    addForm.message = "你好，我想加你为好友。";
    addFriendDialogVisible.value = false;
  } catch (error) {
    ElMessage.error(error?.message || "好友申请发送失败");
  } finally {
    applying.value = false;
  }
}

async function handleApplication(action, application) {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  const key = `${action}-${application.contact_id}`;
  actionLoading.value = key;

  try {
    const result =
      action === "pass"
        ? await passContactApply(user.uuid, application.contact_id)
        : await refuseContactApply(user.uuid, application.contact_id);

    ElMessage.success(result.message || (action === "pass" ? "已通过好友申请" : "已拒绝好友申请"));
    removeApplicationLocally(application.contact_id);
    if (action === "pass") {
      rememberLocalFriendAdd(application);
      selectAvailableFriend();
    }
    await refreshAll();
  } catch (error) {
    ElMessage.error(error?.message || "处理好友申请失败");
  } finally {
    actionLoading.value = "";
  }
}

async function blockApplication(application) {
  const user = await ensureUser();
  if (!user || !application?.contact_id) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定拉黑「${application.contact_name || application.contact_id}」的好友申请吗？拉黑后对方不能再次向你发送申请。`, "拉黑申请", {
      confirmButtonText: "拉黑",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = `black-apply-${application.contact_id}`;

  try {
    const result = await blackApply(user.uuid, application.contact_id);
    ElMessage.success(result.message || "已拉黑该申请");
    removeApplicationLocally(application.contact_id);
    activeSection.value = "applications";
    await refreshAll();
  } catch (error) {
    ElMessage.error(error?.message || "拉黑申请失败");
  } finally {
    actionLoading.value = "";
  }
}

function showFriendList() {
  activeSection.value = "friends";
}

function showApplications() {
  activeSection.value = "applications";
}

function openAddFriendDialog() {
  addFriendDialogVisible.value = true;
}

function closeAddFriendDialog() {
  addFriendDialogVisible.value = false;
}

async function startChat() {
  const user = await ensureUser();
  const friend = selectedFriend.value;

  if (!user || !friend) {
    return;
  }

  actionLoading.value = `chat-${friend.user_id}`;

  try {
    const allowed = await checkOpenSessionAllowed(user.uuid, friend.user_id);
    if (allowed.data !== true) {
      throw new Error(allowed.message || "暂时无法开始聊天");
    }

    const result = await openSession(user.uuid, friend.user_id);
    ElMessage.success(result.message || "已打开聊天");
    await router.push({
      name: "messages",
      query: {
        type: "user",
        id: friend.user_id,
        session_id: result.data || "",
        name: friend.user_name || selectedDetail.value?.contact_name || "",
        avatar: friend.avatar || selectedDetail.value?.contact_avatar || "",
      },
    });
  } catch (error) {
    ElMessage.error(error?.message || "打开聊天失败");
  } finally {
    actionLoading.value = "";
  }
}

async function removeSelectedFriend() {
  const user = await ensureUser();
  const friend = selectedFriend.value;

  if (!user || !friend) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定删除好友「${friend.user_name}」吗？`, "删除好友", {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = `delete-${friend.user_id}`;

  try {
    const result = await deleteContact(user.uuid, friend.user_id);
    ElMessage.success(result.message || "已删除好友");
    rememberLocalFriendRemoval(friend.user_id);
    selectAvailableFriend();
    await refreshAll();
  } catch (error) {
    ElMessage.error(error?.message || "删除好友失败");
  } finally {
    actionLoading.value = "";
  }
}

async function blockSelectedFriend() {
  const user = await ensureUser();
  const friend = selectedFriend.value;

  if (!user || !friend) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定拉黑好友「${friend.user_name}」吗？`, "拉黑好友", {
      confirmButtonText: "拉黑",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = `black-${friend.user_id}`;

  try {
    const result = await blackContact(user.uuid, friend.user_id);
    contactStatusOverrides.value = {
      ...contactStatusOverrides.value,
      [friend.user_id]: 2,
    };
    ElMessage.success(result.message || "已拉黑好友");
    await refreshAll();
  } catch (error) {
    ElMessage.error(error?.message || "拉黑好友失败");
  } finally {
    actionLoading.value = "";
  }
}

async function unblockSelectedFriend() {
  const user = await ensureUser();
  const friend = selectedFriend.value;

  if (!user || !friend) {
    return;
  }

  actionLoading.value = `cancel-black-${friend.user_id}`;

  try {
    const result = await cancelBlackContact(user.uuid, friend.user_id);
    contactStatusOverrides.value = {
      ...contactStatusOverrides.value,
      [friend.user_id]: 0,
    };
    ElMessage.success(result.message || "已解除拉黑");
    await refreshAll();
  } catch (error) {
    ElMessage.error(error?.message || "解除拉黑失败");
  } finally {
    actionLoading.value = "";
  }
}

async function goHome() {
  await router.push("/");
}

async function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  await router.push("/auth");
}

onMounted(() => {
  refreshAll();
});
</script>

<template>
  <main class="friends-page">
    <section class="page-shell friends-page__shell">
      <header class="glass-card friends-navbar">
        <div class="friends-navbar__brand">
          <button type="button" class="friends-navbar__back" @click="goHome">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <span class="friends-navbar__logo">K</span>
          <div>
            <p class="friends-navbar__eyebrow">KamaChat</p>
            <h1 class="friends-navbar__title">我的好友</h1>
          </div>
        </div>

        <div class="friends-navbar__actions">
          <el-button plain @click="showApplications">
            <el-icon><CirclePlus /></el-icon>
            <span>好友申请 {{ applications.length }}</span>
          </el-button>
          <el-button type="primary" @click="openAddFriendDialog">
            <el-icon><UserFilled /></el-icon>
            <span>添加好友</span>
          </el-button>
          <el-button plain @click="refreshAll">
            <el-icon><Refresh /></el-icon>
            <span>刷新</span>
          </el-button>
          <el-button type="primary" plain @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </el-button>
        </div>
      </header>

      <section class="friends-hero-card">
        <div>
          <p class="friends-hero-card__eyebrow">好友中心</p>
          <h2 class="friends-hero-card__title">管理好友、处理申请，随时开启聊天。</h2>
          <p class="friends-hero-card__copy">在这里查看好友，也可以处理新的申请。</p>
        </div>

        <div class="friends-stats-grid">
          <article v-for="item in friendStats" :key="item.label" class="friends-stat" :class="`is-${item.tone}`">
            <span class="friends-stat__icon">
              <el-icon><component :is="item.icon" /></el-icon>
            </span>
            <div>
              <p class="friends-stat__label">{{ item.label }}</p>
              <p class="friends-stat__value">{{ item.value }}</p>
            </div>
          </article>
        </div>
      </section>

      <section class="friends-section-switch" :data-mode="activeSection">
        <span class="friends-section-switch__thumb"></span>
        <button
          type="button"
          class="friends-section-switch__button"
          :class="{ 'is-active': activeSection === 'friends' }"
          @click="showFriendList"
        >
          <el-icon><UserFilled /></el-icon>
          <span>好友列表</span>
        </button>
        <button
          type="button"
          class="friends-section-switch__button"
          :class="{ 'is-active': activeSection === 'applications' }"
          @click="showApplications"
        >
          <el-icon><CirclePlus /></el-icon>
          <span>好友申请</span>
          <small>{{ applications.length }}</small>
        </button>
      </section>

      <section class="friends-layout" :class="{ 'is-applications': activeSection === 'applications' }">
        <aside v-if="activeSection === 'friends'" class="friends-sidebar">
          <section class="glass-card friends-list-card">
            <header class="friends-panel-heading">
              <div>
                <p class="friends-panel-heading__eyebrow">好友</p>
                <h2 class="friends-panel-heading__title">好友列表</h2>
              </div>
              <span class="friends-panel-heading__badge">{{ filteredFriends.length }}</span>
            </header>

            <el-input v-model="keyword" class="friends-search" placeholder="搜索昵称或账号" clearable>
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>

            <div v-loading="loading" class="friend-list">
              <el-empty v-if="!filteredFriends.length && !loading" description="还没有好友" />

              <button
                v-for="friend in filteredFriends"
                :key="friend.user_id"
                type="button"
                class="friend-card"
                :class="{ 'is-active': selectedFriendId === friend.user_id }"
                @click="loadFriendDetail(friend.user_id)"
              >
                <el-avatar :size="48" :src="getAvatarUrl(friend.avatar)" class="friend-card__avatar">
                  {{ getInitials(friend.user_name) }}
                </el-avatar>
                <span class="friend-card__body">
                  <strong>{{ friend.user_name || "未命名好友" }}</strong>
                  <small>好友</small>
                </span>
                <span class="friend-card__status" :class="contactStatusClass(getFriendStatus(friend))">
                  {{ contactStatusText(getFriendStatus(friend)) }}
                </span>
              </button>
            </div>
          </section>
        </aside>

        <section class="friends-main">
          <section v-if="activeSection === 'friends'" v-loading="detailLoading" class="glass-card friend-detail-card">
            <template v-if="selectedFriend">
              <div class="friend-detail-card__banner">
                <el-avatar :size="96" :src="getAvatarUrl(selectedFriend.avatar)" class="friend-detail-card__avatar">
                  {{ getInitials(selectedFriend.user_name) }}
                </el-avatar>
                <div class="friend-detail-card__identity">
                  <p class="friend-detail-card__eyebrow">好友信息</p>
                  <h2>{{ selectedDetail?.contact_name || selectedFriend.user_name }}</h2>
                  <p>好友</p>
                </div>
                <span class="friend-detail-card__status" :class="contactStatusClass(selectedFriendStatus)">
                  <span></span>
                  {{ contactStatusText(selectedFriendStatus) }}
                </span>
              </div>

              <p class="friend-signature">
                {{ selectedDetail?.contact_signature || "这个好友还没有设置个性签名。" }}
              </p>

              <div class="friend-info-grid">
                <article v-for="item in detailCards" :key="item.label" class="friend-info-item">
                  <span class="friend-info-item__icon">
                    <el-icon><component :is="item.icon" /></el-icon>
                  </span>
                  <div>
                    <p>{{ item.label }}</p>
                    <strong>{{ item.value }}</strong>
                  </div>
                </article>
                <article class="friend-info-item">
                  <span class="friend-info-item__icon">
                    <el-icon><Lock /></el-icon>
                  </span>
                  <div>
                    <p>生日</p>
                    <strong>{{ formatBirthday(selectedDetail?.contact_birthday) }}</strong>
                  </div>
                </article>
              </div>

              <div class="friend-actions">
                <el-button
                  type="primary"
                  :disabled="Number(selectedFriendStatus) === 1 || Number(selectedFriendStatus) === 2"
                  :loading="actionLoading === `chat-${selectedFriend.user_id}`"
                  @click="startChat"
                >
                  <el-icon><Message /></el-icon>
                  <span>发起聊天</span>
                </el-button>
                <el-button v-if="!selectedFriendBlockedByMe" plain :loading="actionLoading === `black-${selectedFriend.user_id}`" @click="blockSelectedFriend">
                  <el-icon><Warning /></el-icon>
                  <span>拉黑</span>
                </el-button>
                <el-button
                  v-else
                  plain
                  :loading="actionLoading === `cancel-black-${selectedFriend.user_id}`"
                  @click="unblockSelectedFriend"
                >
                  <el-icon><Unlock /></el-icon>
                  <span>解除拉黑</span>
                </el-button>
                <el-button plain :loading="actionLoading === `delete-${selectedFriend.user_id}`" @click="removeSelectedFriend">
                  <el-icon><Delete /></el-icon>
                  <span>删除好友</span>
                </el-button>
              </div>
            </template>

            <el-empty v-else description="请选择一个好友查看信息" />
          </section>

          <section v-else class="glass-card applications-card">
            <header class="friends-panel-heading">
              <div>
                <p class="friends-panel-heading__eyebrow">好友申请</p>
                <h2 class="friends-panel-heading__title">待处理</h2>
              </div>
              <div class="friends-panel-heading__actions">
                <span class="friends-panel-heading__badge">{{ applications.length }}</span>
                <el-button size="small" text @click="showFriendList">返回列表</el-button>
              </div>
            </header>

            <div v-loading="applicationLoading" class="application-list">
              <el-empty v-if="!applications.length && !applicationLoading" description="暂无新的好友申请" />

              <article v-for="application in applications" :key="application.contact_id" class="application-item">
                <el-avatar :size="46" :src="getAvatarUrl(application.contact_avatar)" class="application-item__avatar">
                  {{ getInitials(application.contact_name) }}
                </el-avatar>

                <div class="application-item__body">
                  <h3>{{ application.contact_name || "未命名用户" }}</h3>
                  <p>{{ application.message || "申请添加你为好友" }}</p>
                </div>

                <div class="application-item__actions">
                  <el-button
                    size="small"
                    type="primary"
                    title="通过申请"
                    :loading="actionLoading === `pass-${application.contact_id}`"
                    @click="handleApplication('pass', application)"
                  >
                    <el-icon><Check /></el-icon>
                  </el-button>
                  <el-button
                    size="small"
                    plain
                    title="拒绝申请"
                    :loading="actionLoading === `refuse-${application.contact_id}`"
                    @click="handleApplication('refuse', application)"
                  >
                    <el-icon><Close /></el-icon>
                  </el-button>
                  <el-button
                    size="small"
                    plain
                    class="application-item__block"
                    title="拉黑申请"
                    :loading="actionLoading === `black-apply-${application.contact_id}`"
                    @click="blockApplication(application)"
                  >
                    <el-icon><Warning /></el-icon>
                  </el-button>
                </div>
              </article>
            </div>
          </section>
        </section>
      </section>

      <el-dialog
        v-model="addFriendDialogVisible"
        title="发送好友申请"
        width="520px"
        class="friends-dialog"
        :teleported="false"
        @closed="closeAddFriendDialog"
      >
        <div class="add-friend-form">
          <el-input v-model="addForm.contactId" placeholder="输入好友账号" clearable />
          <el-input
            v-model="addForm.message"
            type="textarea"
            :rows="3"
            maxlength="40"
            show-word-limit
            placeholder="申请理由"
          />
        </div>

        <template #footer>
          <div class="dialog-actions">
            <el-button @click="closeAddFriendDialog">取消</el-button>
            <el-button type="primary" :loading="applying" @click="submitAddFriend">
              <el-icon><CirclePlus /></el-icon>
              <span>发送申请</span>
            </el-button>
          </div>
        </template>
      </el-dialog>

      <section class="glass-card friends-footer-card">
        <div>
          <strong>回到聊天首页</strong>
          <span>也可以去看消息和群聊。</span>
        </div>
        <el-button size="large" @click="goHome">
          <el-icon><House /></el-icon>
          <span>返回首页</span>
        </el-button>
      </section>
    </section>
  </main>
</template>

<style scoped>
.friends-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 14% 12%, rgba(255, 255, 255, 0.86) 0, rgba(255, 255, 255, 0) 27%),
    radial-gradient(circle at 84% 16%, rgba(255, 211, 187, 0.4) 0, rgba(255, 211, 187, 0) 26%),
    linear-gradient(135deg, #fff7f1 0%, #ffe7d8 48%, #f8f1e9 100%);
}

.friends-page__shell {
  display: grid;
  gap: 20px;
  width: min(1180px, calc(100% - 32px));
}

.friends-navbar,
.friends-navbar__brand,
.friends-navbar__actions {
  display: flex;
  align-items: center;
}

.friends-navbar {
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.friends-navbar__brand,
.friends-navbar__actions {
  gap: 14px;
}

.friends-navbar__back {
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

.friends-navbar__logo {
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

.friends-navbar__eyebrow,
.friends-navbar__title {
  margin: 0;
}

.friends-navbar__eyebrow {
  color: var(--kc-muted);
  font-size: 12px;
}

.friends-navbar__title {
  font-size: 18px;
  font-weight: 800;
}

.friends-hero-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 480px);
  gap: 24px;
  padding: 28px;
  border: 1px solid rgba(225, 176, 146, 0.36);
  border-radius: 24px;
  background:
    linear-gradient(145deg, rgba(255, 250, 246, 0.96) 0%, rgba(255, 225, 205, 0.9) 100%),
    #ffffff;
  box-shadow: 0 22px 42px rgba(145, 87, 58, 0.1);
}

.friends-hero-card__eyebrow,
.friends-panel-heading__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.friends-hero-card__title {
  max-width: 640px;
  margin: 0;
  color: #2f211a;
  font-size: 30px;
  line-height: 1.18;
  font-weight: 800;
}

.friends-hero-card__copy {
  max-width: 560px;
  margin: 14px 0 0;
  color: var(--kc-muted);
  line-height: 1.7;
}

.friends-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  align-content: center;
}

.friends-stat {
  display: grid;
  gap: 12px;
  min-height: 124px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.66);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.62);
}

.friends-stat__icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 15px;
  color: #8f4829;
}

.friends-stat.is-peach .friends-stat__icon {
  background: #ffe0cc;
}

.friends-stat.is-rose .friends-stat__icon {
  background: #ffe0e0;
}

.friends-stat.is-mint .friends-stat__icon {
  background: #def7e9;
}

.friends-stat__label,
.friends-stat__value {
  margin: 0;
}

.friends-stat__label {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.friends-stat__value {
  margin-top: 6px;
  color: #2f211a;
  font-size: 24px;
  font-weight: 800;
}

.friends-section-switch {
  position: relative;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: min(560px, 100%);
  padding: 4px;
  border: 1px solid rgba(213, 166, 140, 0.36);
  border-radius: 999px;
  background: rgba(255, 250, 246, 0.82);
  box-shadow: 0 14px 28px rgba(145, 87, 58, 0.08);
}

.friends-section-switch__thumb {
  position: absolute;
  top: 4px;
  left: 4px;
  width: calc(50% - 4px);
  height: calc(100% - 8px);
  border-radius: 999px;
  background: linear-gradient(135deg, #ffc89c 0%, #f1aa75 100%);
  box-shadow: 0 10px 22px rgba(185, 107, 63, 0.16);
  transition: transform 0.25s ease;
}

.friends-section-switch[data-mode="applications"] .friends-section-switch__thumb {
  transform: translateX(calc(100% + 4px));
}

.friends-section-switch__button {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 42px;
  border: 0;
  background: transparent;
  color: #836355;
  font: inherit;
  font-weight: 800;
  cursor: pointer;
}

.friends-section-switch__button.is-active {
  color: #5e3421;
}

.friends-section-switch__button small {
  display: inline-grid;
  place-items: center;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.76);
  color: #9d4d2d;
  font-size: 12px;
}

.friends-layout {
  display: grid;
  grid-template-columns: minmax(300px, 360px) minmax(0, 1fr);
  gap: 20px;
}

.friends-layout.is-applications {
  grid-template-columns: 1fr;
}

.friends-sidebar,
.friends-main {
  display: grid;
  gap: 20px;
  align-content: start;
}

.friends-list-card,
.add-friend-card,
.friend-detail-card,
.applications-card,
.friends-footer-card {
  padding: 22px;
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(18px);
}

.friends-panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.friends-panel-heading__title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
}

.friends-panel-heading__badge {
  min-width: 32px;
  padding: 7px 10px;
  border-radius: 999px;
  background: #fff0e2;
  color: #9d4d2d;
  font-size: 12px;
  font-weight: 800;
  text-align: center;
}

.friends-panel-heading__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.friends-panel-heading__actions :deep(.el-button) {
  color: #9d4d2d;
  font-weight: 800;
}

.friends-search {
  margin-bottom: 14px;
}

.friends-search :deep(.el-input__wrapper),
.add-friend-form :deep(.el-input__wrapper),
.add-friend-form :deep(.el-textarea__inner) {
  min-height: 48px;
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
}

.friends-search :deep(.el-input__wrapper.is-focus),
.add-friend-form :deep(.el-input__wrapper.is-focus),
.add-friend-form :deep(.el-textarea__inner:focus) {
  box-shadow:
    0 0 0 1.5px rgba(185, 104, 61, 0.72) inset,
    0 14px 24px rgba(182, 110, 73, 0.12);
}

.friend-list,
.application-list {
  display: grid;
  gap: 10px;
  min-height: 120px;
}

.friend-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 74px;
  padding: 12px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.friend-card:hover,
.friend-card.is-active {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.34);
  box-shadow: 0 12px 22px rgba(145, 87, 58, 0.08);
}

.friend-card.is-active {
  background: rgba(255, 239, 227, 0.88);
}

.friend-card__avatar,
.friend-detail-card__avatar,
.application-item__avatar {
  background: linear-gradient(135deg, #fff3e8 0%, #f7b17e 100%);
  color: #8f4829;
  font-weight: 800;
}

.friend-card__body {
  min-width: 0;
}

.friend-card__body strong,
.friend-card__body small {
  display: block;
}

.friend-card__body strong {
  overflow: hidden;
  color: #2f211a;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.friend-card__body small {
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 12px;
}

.friend-card__status,
.friend-detail-card__status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 6px 9px;
  border-radius: 999px;
  background: rgba(222, 247, 233, 0.88);
  color: #1f7656;
  font-size: 12px;
  font-weight: 800;
}

.friend-card__status.is-blocked,
.friend-detail-card__status.is-blocked {
  background: rgba(255, 229, 221, 0.94);
  color: #a9472f;
}

.friend-card__status.is-blocked-by,
.friend-detail-card__status.is-blocked-by {
  background: rgba(244, 232, 220, 0.94);
  color: #7d6355;
}

.add-friend-form {
  display: grid;
  gap: 12px;
}

.add-friend-form :deep(.el-button) {
  min-height: 46px;
  border-radius: 14px;
}

.friend-detail-card {
  min-height: 430px;
}

.friend-detail-card__banner {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 18px;
  align-items: center;
  padding: 22px;
  border-radius: 22px;
  background: linear-gradient(145deg, #fff8f1 0%, #ffd9c0 100%);
}

.friend-detail-card__identity {
  min-width: 0;
}

.friend-detail-card__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
}

.friend-detail-card__identity h2,
.friend-detail-card__identity p {
  margin: 0;
}

.friend-detail-card__identity h2 {
  color: #2f211a;
  font-size: 28px;
  line-height: 1.2;
  font-weight: 800;
}

.friend-detail-card__identity p {
  margin-top: 8px;
  color: var(--kc-muted);
  font-weight: 700;
}

.friend-detail-card__status span {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #1f8f65;
}

.friend-detail-card__status.is-blocked span,
.friend-detail-card__status.is-blocked-by span {
  background: #bc5a34;
}

.friend-signature {
  min-height: 56px;
  margin: 18px 0 0;
  padding: 16px 18px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
  color: #725444;
  line-height: 1.7;
}

.friend-info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.friend-info-item {
  display: flex;
  gap: 12px;
  align-items: center;
  min-height: 82px;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
}

.friend-info-item__icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  background: #fff0e2;
  color: #9d4d2d;
}

.friend-info-item p,
.friend-info-item strong {
  margin: 0;
}

.friend-info-item p {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.friend-info-item strong {
  display: block;
  margin-top: 5px;
  overflow-wrap: anywhere;
  color: #2f211a;
}

.friend-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 18px;
}

.friend-actions :deep(.el-button) {
  min-height: 44px;
  border-radius: 14px;
}

.application-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
}

.application-item__body {
  min-width: 0;
}

.application-item__body h3,
.application-item__body p {
  margin: 0;
}

.application-item__body h3 {
  color: #2f211a;
  font-size: 15px;
  font-weight: 800;
}

.application-item__body p {
  margin-top: 6px;
  color: var(--kc-muted);
  line-height: 1.6;
}

.application-item__actions {
  display: flex;
  gap: 8px;
}

.application-item__actions :deep(.el-button) {
  width: 34px;
  height: 34px;
  padding: 0;
  border-radius: 12px;
}

.application-item__actions :deep(.application-item__block) {
  color: #a9472f;
  border-color: rgba(188, 90, 52, 0.28);
  background: rgba(255, 238, 231, 0.72);
}

.friends-footer-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.friends-footer-card strong,
.friends-footer-card span {
  display: block;
}

.friends-footer-card strong {
  color: #2f211a;
}

.friends-footer-card span {
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 13px;
}

.friends-footer-card :deep(.el-button) {
  border-radius: 14px;
}

.friends-page :deep(.friends-dialog) {
  overflow: hidden;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 28px 70px rgba(91, 53, 34, 0.18);
}

.friends-page :deep(.friends-dialog .el-dialog__header) {
  padding: 22px 24px 12px;
  margin: 0;
}

.friends-page :deep(.friends-dialog .el-dialog__title) {
  color: #2f211a;
  font-weight: 800;
}

.friends-page :deep(.friends-dialog .el-dialog__body) {
  padding: 12px 24px 22px;
}

.friends-page :deep(.friends-dialog .el-dialog__footer) {
  padding: 0 24px 24px;
}

.dialog-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: flex-end;
}

.dialog-actions :deep(.el-button) {
  min-height: 42px;
  border-radius: 14px;
}

@media (max-width: 1040px) {
  .friends-hero-card,
  .friends-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .friends-navbar,
  .friends-navbar__actions,
  .friends-footer-card {
    align-items: stretch;
    flex-direction: column;
  }

  .friends-stats-grid,
  .friend-info-grid {
    grid-template-columns: 1fr;
  }

  .friend-detail-card__banner,
  .application-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .friend-detail-card__status,
  .application-item__actions {
    grid-column: 2;
    justify-self: start;
  }

  .friends-footer-card :deep(.el-button) {
    width: 100%;
  }
}

@media (max-width: 560px) {
  .friends-page__shell {
    width: min(100% - 20px, 1180px);
  }

  .friends-navbar,
  .friends-hero-card,
  .friends-list-card,
  .add-friend-card,
  .friend-detail-card,
  .applications-card,
  .friends-footer-card {
    padding: 18px;
  }

  .friend-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .friend-card__status {
    grid-column: 2;
    justify-self: start;
  }

  .friend-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
