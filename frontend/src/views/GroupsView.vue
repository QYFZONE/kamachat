<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  ChatDotRound,
  Check,
  Delete,
  Edit,
  House,
  Lock,
  Message,
  Refresh,
  Setting,
  SwitchButton,
  User,
  UserFilled,
  Warning,
} from "@element-plus/icons-vue";

import { resolveAssetUrl } from "../api/http";
import {
  dismissGroup,
  getCreatedGroups,
  getGroupInfo,
  getGroupMemberList,
  getJoinedGroups,
  getUserInfo,
  leaveGroup,
  openSession,
  removeGroupMembers,
  updateGroupInfo,
} from "../api/user";
import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser } from "../utils/storage";

const router = useRouter();
const currentUser = ref(getStoredUser() || {});
const activeGroupType = ref("created");
const selectedGroupId = ref("");
const loading = ref(false);
const detailLoading = ref(false);
const actionLoading = ref("");
const memberDetailLoading = ref(false);
const createdGroups = ref([]);
const joinedGroups = ref([]);
const members = ref([]);
const groupDetail = ref(null);
const selectedMemberId = ref("");
const selectedMemberDetail = ref(null);
const membersVisible = ref(false);
const editDialogVisible = ref(false);
const memberDetailDialogVisible = ref(false);

const editFormRef = ref();

const editForm = reactive({
  name: "",
  notice: "",
  addMode: 0,
});

const groupRules = {
  name: [
    { required: true, message: "群名称不能为空", trigger: "blur" },
    { min: 2, max: 20, message: "群名称为 2 到 20 个字", trigger: "blur" },
  ],
  notice: [{ max: 120, message: "群公告不能超过 120 个字", trigger: "blur" }],
};

const activeGroups = computed(() => (activeGroupType.value === "created" ? createdGroups.value : joinedGroups.value));
const selectedGroup = computed(() => activeGroups.value.find((group) => group.group_id === selectedGroupId.value) || null);
const isOwner = computed(() => groupDetail.value?.owner_id === currentUser.value?.uuid);
const selectedMember = computed(() => members.value.find((member) => member.user_id === selectedMemberId.value) || null);

const groupStats = computed(() => [
  {
    label: "我创建的",
    value: createdGroups.value.length,
    icon: UserFilled,
    tone: "peach",
  },
  {
    label: "我加入的",
    value: joinedGroups.value.length,
    icon: ChatDotRound,
    tone: "rose",
  },
  {
    label: "成员",
    value: groupDetail.value?.member_cnt || members.value.length || 0,
    icon: User,
    tone: "mint",
  },
]);

const groupMeta = computed(() => [
  {
    label: "成员人数",
    value: groupDetail.value?.member_cnt || members.value.length || 0,
    icon: User,
  },
  {
    label: "加群方式",
    value: addModeText(groupDetail.value?.add_mode),
    icon: Setting,
  },
  {
    label: "群情况",
    value: groupStatusText(groupDetail.value?.status),
    icon: Check,
  },
  {
    label: "我的角色",
    value: isOwner.value ? "群主" : "成员",
    icon: UserFilled,
  },
]);

function getInitials(name) {
  return String(name || "群").trim().slice(0, 1).toUpperCase();
}

function getAvatarUrl(avatar) {
  return resolveAssetUrl(avatar || "");
}

function addModeText(value) {
  return Number(value) === 1 ? "需要审核" : "直接加入";
}

function groupStatusText(value) {
  const status = Number(value || 0);

  if (status === 1) {
    return "已禁用";
  }

  if (status === 2) {
    return "已解散";
  }

  return "正常";
}

function normalizeGroups(data) {
  return Array.isArray(data) ? data : [];
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

function syncEditForm(detail) {
  editForm.name = detail?.name || "";
  editForm.notice = detail?.notice || "";
  editForm.addMode = Number(detail?.add_mode || 0);
}

function closeMemberDetail() {
  selectedMemberId.value = "";
  selectedMemberDetail.value = null;
  memberDetailDialogVisible.value = false;
}

function closeMembers() {
  membersVisible.value = false;
  closeMemberDetail();
}

function showMembers() {
  if (!groupDetail.value && !selectedGroup.value) {
    ElMessage.warning("请先选择一个群聊");
    return;
  }

  membersVisible.value = true;
}

function openEditDialog() {
  if (!isOwner.value || !groupDetail.value) {
    return;
  }

  syncEditForm(groupDetail.value);
  editDialogVisible.value = true;
}

function closeEditDialog() {
  editDialogVisible.value = false;
  syncEditForm(groupDetail.value);
}

async function loadGroups() {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  loading.value = true;

  try {
    const [createdResult, joinedResult] = await Promise.allSettled([
      getCreatedGroups(user.uuid),
      getJoinedGroups(user.uuid),
    ]);

    createdGroups.value = createdResult.status === "fulfilled" ? normalizeGroups(createdResult.value.data) : [];
    joinedGroups.value = joinedResult.status === "fulfilled" ? normalizeGroups(joinedResult.value.data) : [];

    const nextList = activeGroups.value;
    if (!selectedGroupId.value || !nextList.some((group) => group.group_id === selectedGroupId.value)) {
      selectedGroupId.value = nextList[0]?.group_id || "";
    }

    if (selectedGroupId.value) {
      await loadGroupDetail(selectedGroupId.value);
    } else {
      groupDetail.value = null;
      members.value = [];
      closeMembers();
      editDialogVisible.value = false;
      syncEditForm(null);
    }
  } catch (error) {
    ElMessage.error(error?.message || "群聊列表加载失败");
  } finally {
    loading.value = false;
  }
}

async function loadGroupDetail(groupId, options = {}) {
  const keepMembersOpen = options.keepMembersOpen === true;

  if (!groupId) {
    groupDetail.value = null;
    members.value = [];
    closeMembers();
    editDialogVisible.value = false;
    syncEditForm(null);
    return;
  }

  selectedGroupId.value = groupId;
  editDialogVisible.value = false;
  if (keepMembersOpen) {
    closeMemberDetail();
  } else {
    closeMembers();
  }
  detailLoading.value = true;

  try {
    const [detailResult, membersResult] = await Promise.all([
      getGroupInfo(groupId),
      getGroupMemberList(groupId),
    ]);

    groupDetail.value = detailResult.data || null;
    members.value = Array.isArray(membersResult.data) ? membersResult.data : [];
    syncEditForm(groupDetail.value);
  } catch (error) {
    groupDetail.value = null;
    members.value = [];
    closeMembers();
    editDialogVisible.value = false;
    syncEditForm(null);
    ElMessage.warning(error?.message || "群聊详情加载失败");
  } finally {
    detailLoading.value = false;
  }
}

async function submitEditGroup() {
  const user = await ensureUser();
  if (!user || !groupDetail.value) {
    return;
  }

  try {
    await editFormRef.value?.validate();
    actionLoading.value = "edit";

    const result = await updateGroupInfo({
      user_id: user.uuid,
      uuid: groupDetail.value.uuid,
      name: editForm.name.trim(),
      notice: editForm.notice.trim(),
      add_mode: editForm.addMode,
      avatar: groupDetail.value.avatar || "",
    });

    ElMessage.success(result.message || "群信息已保存");
    editDialogVisible.value = false;
    await loadGroups();
  } catch (error) {
    ElMessage.error(error?.message || "保存群信息失败");
  } finally {
    actionLoading.value = "";
  }
}

async function loadMemberDetail(member) {
  if (!member?.user_id) {
    selectedMemberId.value = "";
    selectedMemberDetail.value = null;
    return;
  }

  selectedMemberId.value = member.user_id;
  memberDetailDialogVisible.value = true;
  memberDetailLoading.value = true;

  try {
    const result = await getUserInfo(member.user_id);
    selectedMemberDetail.value = result.data || null;
  } catch (error) {
    selectedMemberDetail.value = null;
    ElMessage.warning(error?.message || "成员详情加载失败");
  } finally {
    memberDetailLoading.value = false;
  }
}

async function kickMember(member) {
  const user = await ensureUser();

  if (!user || !groupDetail.value || !member?.user_id) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定将「${member.nickname}」移出群聊吗？`, "移出群成员", {
      confirmButtonText: "移出",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = `kick-${member.user_id}`;

  try {
    const result = await removeGroupMembers({
      group_id: groupDetail.value.uuid,
      owner_id: user.uuid,
      uuid_list: [member.user_id],
    });
    ElMessage.success(result.message || "已移出群成员");
    closeMemberDetail();
    membersVisible.value = true;
    await loadGroupDetail(groupDetail.value.uuid, { keepMembersOpen: true });
  } catch (error) {
    ElMessage.error(error?.message || "移出群成员失败");
  } finally {
    actionLoading.value = "";
  }
}

async function openGroupChat() {
  const user = await ensureUser();
  const groupId = groupDetail.value?.uuid || selectedGroup.value?.group_id;

  if (!user || !groupId) {
    return;
  }

  actionLoading.value = "chat";

  try {
    const result = await openSession(user.uuid, groupId);
    ElMessage.success(result.message || "已打开群聊");
    await router.push({
      name: "messages",
      query: {
        type: "group",
        id: groupId,
        session_id: result.data || "",
        name: groupDetail.value?.name || selectedGroup.value?.group_name || "",
        avatar: groupDetail.value?.avatar || selectedGroup.value?.avatar || "",
      },
    });
  } catch (error) {
    ElMessage.error(error?.message || "打开群聊失败");
  } finally {
    actionLoading.value = "";
  }
}

async function leaveSelectedGroup() {
  const user = await ensureUser();
  const group = groupDetail.value;

  if (!user || !group) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定退出群聊「${group.name}」吗？`, "退出群聊", {
      confirmButtonText: "退出",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = "leave";

  try {
    const result = await leaveGroup(user.uuid, group.uuid);
    ElMessage.success(result.message || "已退出群聊");
    selectedGroupId.value = "";
    await loadGroups();
  } catch (error) {
    ElMessage.error(error?.message || "退群失败");
  } finally {
    actionLoading.value = "";
  }
}

async function dismissSelectedGroup() {
  const user = await ensureUser();
  const group = groupDetail.value;

  if (!user || !group) {
    return;
  }

  try {
    await ElMessageBox.confirm(`确定解散群聊「${group.name}」吗？此操作不可恢复。`, "解散群聊", {
      confirmButtonText: "解散",
      cancelButtonText: "取消",
      type: "warning",
    });
  } catch (_error) {
    return;
  }

  actionLoading.value = "dismiss";

  try {
    const result = await dismissGroup(user.uuid, group.uuid);
    ElMessage.success(result.message || "群聊已解散");
    selectedGroupId.value = "";
    await loadGroups();
  } catch (error) {
    ElMessage.error(error?.message || "解散群聊失败");
  } finally {
    actionLoading.value = "";
  }
}

async function goHome() {
  await router.push("/");
}

async function goCreateGroup() {
  await router.push({ name: "createGroup" });
}

async function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  await router.push("/auth");
}

watch(activeGroupType, async () => {
  selectedGroupId.value = activeGroups.value[0]?.group_id || "";
  await loadGroupDetail(selectedGroupId.value);
});

onMounted(() => {
  loadGroups();
});
</script>

<template>
  <main class="groups-page">
    <section class="page-shell groups-page__shell">
      <header class="glass-card groups-navbar">
        <div class="groups-navbar__brand">
          <button type="button" class="groups-navbar__back" @click="goHome">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <span class="groups-navbar__logo">K</span>
          <div>
            <p class="groups-navbar__eyebrow">KamaChat</p>
            <h1 class="groups-navbar__title">我的群聊</h1>
          </div>
        </div>

        <div class="groups-navbar__actions">
          <el-button plain @click="loadGroups">
            <el-icon><Refresh /></el-icon>
            <span>刷新</span>
          </el-button>
          <el-button type="primary" @click="goCreateGroup">
            <el-icon><Edit /></el-icon>
            <span>创建群聊</span>
          </el-button>
          <el-button type="primary" plain @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </el-button>
        </div>
      </header>

      <section class="groups-hero-card">
        <div>
          <p class="groups-hero-card__eyebrow">群聊中心</p>
          <h2 class="groups-hero-card__title">区分我创建的群和我加入的群，管理更清楚。</h2>
          <p class="groups-hero-card__copy">查看群成员，或继续群里的聊天。</p>
        </div>

        <div class="groups-stats-grid">
          <article v-for="item in groupStats" :key="item.label" class="groups-stat" :class="`is-${item.tone}`">
            <span class="groups-stat__icon">
              <el-icon><component :is="item.icon" /></el-icon>
            </span>
            <div>
              <p class="groups-stat__label">{{ item.label }}</p>
              <p class="groups-stat__value">{{ item.value }}</p>
            </div>
          </article>
        </div>
      </section>

      <section class="groups-layout">
        <aside class="groups-sidebar">
          <section class="glass-card groups-list-card">
            <header class="groups-panel-heading">
              <div>
                <p class="groups-panel-heading__eyebrow">群聊</p>
                <h2 class="groups-panel-heading__title">我的群聊</h2>
              </div>
              <span class="groups-panel-heading__badge">{{ activeGroups.length }}</span>
            </header>

            <div class="groups-switch" :data-mode="activeGroupType">
              <span class="groups-switch__thumb"></span>
              <button
                type="button"
                class="groups-switch__button"
                :class="{ 'is-active': activeGroupType === 'created' }"
                @click="activeGroupType = 'created'"
              >
                我创建的
              </button>
              <button
                type="button"
                class="groups-switch__button"
                :class="{ 'is-active': activeGroupType === 'joined' }"
                @click="activeGroupType = 'joined'"
              >
                我加入的
              </button>
            </div>

            <div v-loading="loading" class="group-list">
              <el-empty v-if="!activeGroups.length && !loading" description="暂无群聊" />

              <button
                v-for="group in activeGroups"
                :key="group.group_id"
                type="button"
                class="group-card"
                :class="{ 'is-active': selectedGroupId === group.group_id }"
                @click="loadGroupDetail(group.group_id)"
              >
                <el-avatar :size="48" :src="getAvatarUrl(group.avatar)" class="group-card__avatar">
                  {{ getInitials(group.group_name) }}
                </el-avatar>
                <span class="group-card__body">
                  <strong>{{ group.group_name || "未命名群聊" }}</strong>
                  <small>{{ activeGroupType === "created" ? "我创建的群" : "我加入的群" }}</small>
                </span>
                <span class="group-card__tag">{{ activeGroupType === "created" ? "群主" : "成员" }}</span>
              </button>
            </div>
          </section>
        </aside>

        <section class="groups-main">
          <section v-loading="detailLoading" class="glass-card group-detail-card">
            <template v-if="groupDetail || selectedGroup">
              <div class="group-detail-card__banner">
                <el-avatar :size="96" :src="getAvatarUrl(groupDetail?.avatar || selectedGroup?.avatar)" class="group-detail-card__avatar">
                  {{ getInitials(groupDetail?.name || selectedGroup?.group_name) }}
                </el-avatar>
                <div class="group-detail-card__identity">
                  <p class="group-detail-card__eyebrow">{{ isOwner ? "我创建的群" : "我加入的群" }}</p>
                  <h2>{{ groupDetail?.name || selectedGroup?.group_name }}</h2>
                  <p>群聊</p>
                </div>
                <span class="group-detail-card__status">
                  <span></span>
                  {{ groupStatusText(groupDetail?.status) }}
                </span>
              </div>

              <p class="group-notice">
                {{ groupDetail?.notice || "这个群聊还没有设置群公告。" }}
              </p>

              <div class="group-info-grid">
                <article v-for="item in groupMeta" :key="item.label" class="group-info-item">
                  <span class="group-info-item__icon">
                    <el-icon><component :is="item.icon" /></el-icon>
                  </span>
                  <div>
                    <p>{{ item.label }}</p>
                    <strong>{{ item.value }}</strong>
                  </div>
                </article>
              </div>

              <div class="group-actions">
                <el-button type="primary" :loading="actionLoading === 'chat'" @click="openGroupChat">
                  <el-icon><Message /></el-icon>
                  <span>打开群聊</span>
                </el-button>
                <el-button plain @click="membersVisible ? closeMembers() : showMembers()">
                  <el-icon><User /></el-icon>
                  <span>{{ membersVisible ? "收起群成员" : "查看群成员" }}</span>
                </el-button>
                <el-button v-if="isOwner" plain @click="openEditDialog">
                  <el-icon><Edit /></el-icon>
                  <span>编辑群信息</span>
                </el-button>
                <el-button v-if="!isOwner" plain :loading="actionLoading === 'leave'" @click="leaveSelectedGroup">
                  <el-icon><Warning /></el-icon>
                  <span>退出群聊</span>
                </el-button>
                <el-button v-else plain :loading="actionLoading === 'dismiss'" @click="dismissSelectedGroup">
                  <el-icon><Warning /></el-icon>
                  <span>解散群聊</span>
                </el-button>
              </div>
            </template>

            <el-empty v-else description="请选择一个群聊查看信息" />
          </section>

          <section v-if="membersVisible" class="groups-secondary-grid">
            <section class="glass-card members-card">
              <header class="groups-panel-heading">
                <div>
                  <p class="groups-panel-heading__eyebrow">群成员</p>
                  <h2 class="groups-panel-heading__title">成员列表</h2>
                </div>
                <div class="groups-panel-heading__actions">
                  <span class="groups-panel-heading__badge">{{ members.length }}</span>
                  <el-button size="small" text @click="closeMembers">收起</el-button>
                </div>
              </header>

              <div class="member-list">
                <el-empty v-if="!members.length" description="暂无成员信息" />

                <article
                  v-for="member in members"
                  :key="member.user_id"
                  class="member-item"
                  :class="{ 'is-active': selectedMemberId === member.user_id }"
                >
                  <button type="button" class="member-item__profile" @click="loadMemberDetail(member)">
                    <el-avatar :size="42" :src="getAvatarUrl(member.avatar)" class="member-item__avatar">
                      {{ getInitials(member.nickname) }}
                    </el-avatar>
                    <span>
                      <strong>{{ member.nickname || "未命名成员" }}</strong>
                      <small>成员</small>
                    </span>
                  </button>

                  <span v-if="member.user_id === groupDetail?.owner_id" class="member-role">群主</span>
                  <el-button
                    v-else-if="isOwner"
                    size="small"
                    plain
                    :loading="actionLoading === `kick-${member.user_id}`"
                    @click="kickMember(member)"
                  >
                    <el-icon><Delete /></el-icon>
                    <span>移出</span>
                  </el-button>
                </article>
              </div>
            </section>
          </section>
        </section>
      </section>

      <el-dialog
        v-model="memberDetailDialogVisible"
        title="成员信息"
        width="560px"
        class="groups-dialog"
        :teleported="false"
        @closed="closeMemberDetail"
      >
        <div v-loading="memberDetailLoading" class="member-detail-body">
          <template v-if="selectedMember">
            <div class="member-detail-hero">
              <el-avatar
                :size="64"
                :src="getAvatarUrl(selectedMemberDetail?.avatar || selectedMember.avatar)"
                class="member-item__avatar"
              >
                {{ getInitials(selectedMemberDetail?.nickname || selectedMember.nickname) }}
              </el-avatar>
              <div>
                <strong>{{ selectedMemberDetail?.nickname || selectedMember.nickname }}</strong>
                <span>群成员</span>
              </div>
            </div>
            <p class="member-detail-signature">
              {{ selectedMemberDetail?.signature || "这个成员还没有设置个性签名。" }}
            </p>
            <div class="member-detail-grid">
              <div>
                <span>手机号</span>
                <strong>{{ selectedMemberDetail?.telephone || "未填写" }}</strong>
              </div>
              <div>
                <span>邮箱</span>
                <strong>{{ selectedMemberDetail?.email || "未填写" }}</strong>
              </div>
              <div>
                <span>账号情况</span>
                <strong>{{ Number(selectedMemberDetail?.status || 0) === 0 ? "正常" : "已禁用" }}</strong>
              </div>
              <div>
                <span>身份</span>
                <strong>{{ selectedMember.user_id === groupDetail?.owner_id ? "群主" : "成员" }}</strong>
              </div>
            </div>
          </template>

          <el-empty v-else description="点击成员查看信息" />
        </div>
      </el-dialog>

      <el-dialog
        v-model="editDialogVisible"
        title="编辑群信息"
        width="560px"
        class="groups-dialog"
        :teleported="false"
        @closed="closeEditDialog"
      >
        <el-form ref="editFormRef" :model="editForm" :rules="groupRules" label-position="top" class="group-form">
          <el-form-item label="群名称" prop="name">
            <el-input v-model="editForm.name" :disabled="!isOwner" maxlength="20" clearable />
          </el-form-item>
          <el-form-item label="加群方式">
            <el-select v-model="editForm.addMode" :disabled="!isOwner" class="group-form__control">
              <el-option :value="0" label="直接加入" />
              <el-option :value="1" label="需要审核" />
            </el-select>
          </el-form-item>
          <el-form-item label="群公告" prop="notice">
            <el-input
              v-model="editForm.notice"
              :disabled="!isOwner"
              type="textarea"
              :rows="4"
              maxlength="120"
              show-word-limit
            />
          </el-form-item>
        </el-form>

        <template #footer>
          <div class="dialog-actions">
            <el-button @click="closeEditDialog">取消</el-button>
            <el-button type="primary" :disabled="!groupDetail" :loading="actionLoading === 'edit'" @click="submitEditGroup">
              <el-icon><Check /></el-icon>
              <span>保存群信息</span>
            </el-button>
          </div>
        </template>
      </el-dialog>

      <section class="glass-card groups-footer-card">
        <div>
          <strong>回到聊天首页</strong>
          <span>也可以去看好友和消息。</span>
        </div>
        <div class="groups-footer-card__actions">
          <el-button size="large" @click="goCreateGroup">
            <el-icon><Edit /></el-icon>
            <span>创建群聊</span>
          </el-button>
          <el-button size="large" @click="goHome">
            <el-icon><House /></el-icon>
            <span>返回首页</span>
          </el-button>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.groups-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 14% 12%, rgba(255, 255, 255, 0.86) 0, rgba(255, 255, 255, 0) 27%),
    radial-gradient(circle at 82% 14%, rgba(255, 214, 188, 0.42) 0, rgba(255, 214, 188, 0) 27%),
    linear-gradient(135deg, #fff8f2 0%, #ffe8d8 48%, #f7f0e8 100%);
}

.groups-page__shell {
  display: grid;
  gap: 20px;
  width: min(1180px, calc(100% - 32px));
}

.groups-navbar,
.groups-navbar__brand,
.groups-navbar__actions {
  display: flex;
  align-items: center;
}

.groups-navbar {
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.groups-navbar__brand,
.groups-navbar__actions {
  gap: 14px;
}

.groups-navbar__back {
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

.groups-navbar__logo {
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

.groups-navbar__eyebrow,
.groups-navbar__title {
  margin: 0;
}

.groups-navbar__eyebrow {
  color: var(--kc-muted);
  font-size: 12px;
}

.groups-navbar__title {
  font-size: 18px;
  font-weight: 800;
}

.groups-hero-card {
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

.groups-hero-card__eyebrow,
.groups-panel-heading__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.groups-hero-card__title {
  max-width: 640px;
  margin: 0;
  color: #2f211a;
  font-size: 30px;
  line-height: 1.18;
  font-weight: 800;
}

.groups-hero-card__copy {
  max-width: 560px;
  margin: 14px 0 0;
  color: var(--kc-muted);
  line-height: 1.7;
}

.groups-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  align-content: center;
}

.groups-stat {
  display: grid;
  gap: 12px;
  min-height: 124px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.66);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.62);
}

.groups-stat__icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 15px;
  color: #8f4829;
}

.groups-stat.is-peach .groups-stat__icon {
  background: #ffe0cc;
}

.groups-stat.is-rose .groups-stat__icon {
  background: #ffe0e0;
}

.groups-stat.is-mint .groups-stat__icon {
  background: #def7e9;
}

.groups-stat__label,
.groups-stat__value {
  margin: 0;
}

.groups-stat__label {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.groups-stat__value {
  margin-top: 6px;
  color: #2f211a;
  font-size: 24px;
  font-weight: 800;
}

.groups-layout {
  display: grid;
  grid-template-columns: minmax(300px, 360px) minmax(0, 1fr);
  gap: 20px;
}

.groups-sidebar,
.groups-main {
  display: grid;
  gap: 20px;
  align-content: start;
}

.groups-list-card,
.create-group-card,
.group-detail-card,
.members-card,
.member-detail-card,
.edit-group-card,
.groups-footer-card {
  padding: 22px;
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(18px);
}

.groups-panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.groups-panel-heading__title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
}

.groups-panel-heading__badge {
  min-width: 32px;
  padding: 7px 10px;
  border-radius: 999px;
  background: #fff0e2;
  color: #9d4d2d;
  font-size: 12px;
  font-weight: 800;
  text-align: center;
}

.groups-panel-heading__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.groups-panel-heading__actions :deep(.el-button) {
  color: #9d4d2d;
  font-weight: 800;
}

.groups-switch {
  position: relative;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 14px;
  padding: 4px;
  border: 1px solid rgba(213, 166, 140, 0.36);
  border-radius: 999px;
  background: rgba(253, 245, 238, 0.95);
}

.groups-switch__thumb {
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

.groups-switch[data-mode="joined"] .groups-switch__thumb {
  transform: translateX(calc(100% + 4px));
}

.groups-switch__button {
  position: relative;
  z-index: 1;
  min-height: 42px;
  border: 0;
  background: transparent;
  color: #836355;
  font: inherit;
  font-weight: 800;
  cursor: pointer;
}

.groups-switch__button.is-active {
  color: #5e3421;
}

.group-list {
  display: grid;
  gap: 10px;
  min-height: 130px;
}

.group-card {
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

.group-card:hover,
.group-card.is-active {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.34);
  box-shadow: 0 12px 22px rgba(145, 87, 58, 0.08);
}

.group-card.is-active {
  background: rgba(255, 239, 227, 0.88);
}

.group-card__avatar,
.group-detail-card__avatar,
.member-item__avatar {
  background: linear-gradient(135deg, #fff3e8 0%, #f7b17e 100%);
  color: #8f4829;
  font-weight: 800;
}

.group-card__body {
  min-width: 0;
}

.group-card__body strong,
.group-card__body small {
  display: block;
}

.group-card__body strong {
  overflow: hidden;
  color: #2f211a;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-card__body small {
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 12px;
}

.group-card__tag,
.group-detail-card__status {
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

.group-form {
  display: grid;
}

.group-form__control {
  width: 100%;
}

.group-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.group-form :deep(.el-form-item__label) {
  padding-bottom: 8px;
  color: #4c3428;
  font-weight: 800;
}

.group-form :deep(.el-input__wrapper),
.group-form :deep(.el-textarea__inner),
.group-form :deep(.el-select__wrapper) {
  min-height: 48px;
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
}

.group-form :deep(.el-button) {
  min-height: 46px;
  border-radius: 14px;
}

.group-detail-card {
  min-height: 420px;
}

.group-detail-card__banner {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 18px;
  align-items: center;
  padding: 22px;
  border-radius: 22px;
  background: linear-gradient(145deg, #fff8f1 0%, #ffd9c0 100%);
}

.group-detail-card__identity {
  min-width: 0;
}

.group-detail-card__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
}

.group-detail-card__identity h2,
.group-detail-card__identity p {
  margin: 0;
}

.group-detail-card__identity h2 {
  color: #2f211a;
  font-size: 28px;
  line-height: 1.2;
  font-weight: 800;
}

.group-detail-card__identity p {
  margin-top: 8px;
  color: var(--kc-muted);
  font-weight: 700;
}

.group-detail-card__status span {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #1f8f65;
}

.group-notice {
  min-height: 56px;
  margin: 18px 0 0;
  padding: 16px 18px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
  color: #725444;
  line-height: 1.7;
}

.group-info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.group-info-item {
  display: flex;
  gap: 12px;
  align-items: center;
  min-height: 82px;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
}

.group-info-item__icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  background: #fff0e2;
  color: #9d4d2d;
}

.group-info-item p,
.group-info-item strong {
  margin: 0;
}

.group-info-item p {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.group-info-item strong {
  display: block;
  margin-top: 5px;
  overflow-wrap: anywhere;
  color: #2f211a;
}

.group-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 18px;
}

.group-actions :deep(.el-button) {
  min-height: 44px;
  border-radius: 14px;
}

.groups-secondary-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 20px;
}

.member-list {
  display: grid;
  gap: 10px;
}

.member-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 12px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.member-item:hover,
.member-item.is-active {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.34);
  box-shadow: 0 12px 22px rgba(145, 87, 58, 0.08);
}

.member-item__profile {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.member-item strong,
.member-item small {
  display: block;
}

.member-item strong {
  color: #2f211a;
  font-size: 14px;
}

.member-item small {
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 12px;
}

.member-role {
  padding: 6px 9px;
  border-radius: 999px;
  background: #fff0e2;
  color: #9d4d2d;
  font-weight: 800;
  font-size: 12px;
}

.member-detail-body {
  min-height: 280px;
}

.member-detail-hero {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 14px;
  border-radius: 18px;
  background: linear-gradient(145deg, #fff8f1 0%, #ffe2ce 100%);
}

.member-detail-hero strong,
.member-detail-hero span {
  display: block;
}

.member-detail-hero strong {
  color: #2f211a;
  font-size: 18px;
}

.member-detail-hero span {
  margin-top: 6px;
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.member-detail-signature {
  margin: 14px 0 0;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 16px;
  background: rgba(255, 250, 246, 0.72);
  color: #725444;
  line-height: 1.7;
}

.member-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.member-detail-grid div {
  min-height: 70px;
  padding: 12px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 16px;
  background: rgba(255, 250, 246, 0.72);
}

.member-detail-grid span,
.member-detail-grid strong {
  display: block;
}

.member-detail-grid span {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.member-detail-grid strong {
  margin-top: 6px;
  overflow-wrap: anywhere;
  color: #2f211a;
}

.groups-footer-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.groups-footer-card strong,
.groups-footer-card span {
  display: block;
}

.groups-footer-card strong {
  color: #2f211a;
}

.groups-footer-card span {
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 13px;
}

.groups-footer-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: flex-end;
}

.groups-footer-card :deep(.el-button) {
  border-radius: 14px;
}

.groups-page :deep(.groups-dialog) {
  overflow: hidden;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 28px 70px rgba(91, 53, 34, 0.18);
}

.groups-page :deep(.groups-dialog .el-dialog__header) {
  padding: 22px 24px 12px;
  margin: 0;
}

.groups-page :deep(.groups-dialog .el-dialog__title) {
  color: #2f211a;
  font-weight: 800;
}

.groups-page :deep(.groups-dialog .el-dialog__body) {
  padding: 12px 24px 22px;
}

.groups-page :deep(.groups-dialog .el-dialog__footer) {
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

@media (max-width: 1080px) {
  .groups-hero-card,
  .groups-layout,
  .groups-secondary-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .groups-navbar,
  .groups-navbar__actions,
  .groups-footer-card {
    align-items: stretch;
    flex-direction: column;
  }

  .groups-stats-grid,
  .group-info-grid,
  .member-detail-grid {
    grid-template-columns: 1fr;
  }

  .group-detail-card__banner {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .group-detail-card__status {
    grid-column: 2;
    justify-self: start;
  }

  .groups-footer-card :deep(.el-button) {
    width: 100%;
  }

  .groups-footer-card__actions {
    justify-content: stretch;
  }
}

@media (max-width: 560px) {
  .groups-page__shell {
    width: min(100% - 20px, 1180px);
  }

  .groups-navbar,
  .groups-hero-card,
  .groups-list-card,
  .create-group-card,
  .group-detail-card,
  .members-card,
  .member-detail-card,
  .edit-group-card,
  .groups-footer-card {
    padding: 18px;
  }

  .group-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .group-card__tag {
    grid-column: 2;
    justify-self: start;
  }

  .group-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
