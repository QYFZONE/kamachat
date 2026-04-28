import { createRouter, createWebHistory } from "vue-router";

import { appMeta, routeTitles } from "../constants/ui-text";
import AuthView from "../views/AuthView.vue";
import CreateGroupView from "../views/CreateGroupView.vue";
import FriendsView from "../views/FriendsView.vue";
import GroupsView from "../views/GroupsView.vue";
import HomeView from "../views/HomeView.vue";
import MessagesView from "../views/MessagesView.vue";
import ProfileView from "../views/ProfileView.vue";
import { getStoredUser } from "../utils/storage";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "home",
      component: HomeView,
      meta: { requiresAuth: true, title: routeTitles.home },
    },
    {
      path: "/my-messages",
      name: "messages",
      component: MessagesView,
      meta: { requiresAuth: true, title: routeTitles.messages },
    },
    {
      path: "/profile",
      name: "profile",
      component: ProfileView,
      meta: { requiresAuth: true, title: routeTitles.profile },
    },
    {
      path: "/friends",
      name: "friends",
      component: FriendsView,
      meta: { requiresAuth: true, title: routeTitles.friends },
    },
    {
      path: "/my-groups",
      name: "groups",
      component: GroupsView,
      meta: { requiresAuth: true, title: routeTitles.groups },
    },
    {
      path: "/create-group",
      name: "createGroup",
      component: CreateGroupView,
      meta: { requiresAuth: true, title: routeTitles.createGroup },
    },
    {
      path: "/auth",
      name: "auth",
      component: AuthView,
      meta: { publicOnly: true, title: routeTitles.auth },
    },
    {
      path: "/:pathMatch(.*)*",
      redirect: "/auth",
    },
  ],
});

router.beforeEach((to) => {
  const user = getStoredUser();

  if (to.meta.requiresAuth && !user) {
    return "/auth";
  }

  if (to.meta.publicOnly && user) {
    return "/";
  }

  return true;
});

router.afterEach((to) => {
  const pageTitle = to.meta.title || appMeta.defaultTitle;
  document.title = `${pageTitle} - ${appMeta.appName}`;
});

export default router;
