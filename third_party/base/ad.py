from datetime import datetime, timezone
from ldap3 import Server, Connection, ALL, SUBTREE, DEREF_ALWAYS
import logging

logger = logging.getLogger(__name__)


class AD:
    def __init__(self, base_dn: str, url: str):
        self.conn = None
        self.base_dn = base_dn
        self.url = url
        self.search_user_filter = '(&(objectclass=person)(sAMAccountName={}))'
        self.ret_attrs = ['SAMAccountName', 'memberOf']
        self.size_limit = 0
        self.time_limit = 0

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.unbind()

    def bind(self, user: str, passwd: str) -> bool:
        server = Server(self.url, connect_timeout=5, use_ssl=False, get_info=ALL)
        conn = Connection(server, user=user, password=passwd)
        if not conn.bind():
            print('登录绑定失败：', conn.result)
            return False
        self.conn = conn
        return True

    def unbind(self):
        if getattr(self, "conn", None):
            try:
                self.conn.unbind()
            except Exception:
                pass
            finally:
                self.conn = None

    def find_or_create_user(self, account_name: str, username: str, ou: str, title: str, default_pwd: str):
        try:
            if self.search_user(account_name):
                print(f"用户 {account_name} 已经存在, 无需创建")
                return None
            else:
                # 执行创建用户的逻辑
                self.create_user(account_name, username, ou, title, default_pwd)
                return None
        except Exception as e:
            error_message = f"用户名 => [{account_name}]，错误信息 => {str(e)}"
            return Exception(error_message)

    def search_user(self, account_name: str):
        formatted_filter = self.search_user_filter.format(account_name)
        self.conn.search(search_base=self.base_dn, search_filter=formatted_filter,
                         search_scope=SUBTREE, attributes=self.ret_attrs,
                         size_limit=self.size_limit, time_limit=self.time_limit,
                         types_only=False, dereference_aliases=DEREF_ALWAYS)

        # 检查获取的值是否为空或者长度是否不为1
        entries_value = getattr(self.conn, 'entries')
        if not entries_value and len(entries_value) != 1:
            return

        return self.conn.entries

    def create_user(self, account_name: str, username: str, ou: str, title: str, default_pwd: str):
        # 如果名字4字以上，一律按照阜新复姓 上官、独孤
        if len(username) > 3:
            compound_family_name = username[:2]
            given_name = username[2:]
        else:
            compound_family_name = username[0]
            given_name = username[1:]

        # 拼接邮箱
        lower_base_dn = self.base_dn.lower()
        mail_domain = lower_base_dn.replace('dc=', '').replace(',', '.')
        mail = account_name + '@' + mail_domain

        # 属性信息
        user_attributes = {
            'sn': compound_family_name,
            'givenName': given_name,
            'name': account_name,
            'mail': mail,
            'title': title,
            'userPrincipalName': mail,
            'SAMAccountName': account_name,
            'displayName': username,
            'objectClass': ['top', 'person', 'organizationalPerson', 'user']
        }

        # 添加用户
        user_dn = 'cn={},ou={},{}'.format(username, ou, self.base_dn)
        result = self.conn.add(dn=user_dn, attributes=user_attributes)

        # 检查添加结果
        if result:
            self.modify_password(user_dn, default_pwd)
            print(f"用户 {account_name} 创建成功")
        else:
            raise Exception(self.conn.result['message'])

    def add_members_to_groups(self, user_dn, group_dn):
        # 不处理错误情况
        return self.conn.extend.microsoft.add_members_to_groups(user_dn, group_dn)

    def modify_password(self, user_dn: str, default_pwd: str):
        self.conn.extend.microsoft.modify_password(user_dn, default_pwd)
        user_attributes = {
            # 用户状态为启用
            'userAccountControl': [('MODIFY_REPLACE', 512)],
            # 用户下次登录需要重置密码
            'pwdLastSet': [('MODIFY_REPLACE', 0)]
        }

        result = self.conn.modify(user_dn, user_attributes)
        if result:
            print("初始化用户密码成功")
        else:
            raise Exception(f"初始化用户密码失败: {self.conn.result['message']}")

    @staticmethod
    def filetime_to_datetime(ft_val):
        """将 AD 100ns FileTime 转换为 UTC datetime"""
        if not ft_val or ft_val in (0, 0x7FFFFFFFFFFFFFFF, 9223372036854775807):
            return None
        if isinstance(ft_val, datetime):
            return ft_val.astimezone(timezone.utc)
        try:
            sec = (int(ft_val) / 10000000) - 11644473600
            return datetime.fromtimestamp(sec, tz=timezone.utc) if sec > 0 else None
        except Exception:
            return None

    def search_user_password_expiry(self, account_name: str):
        """查询指定账号的密码到期情况"""
        attrs = [
            "sAMAccountName", "displayName", "mail",
            "userAccountControl", "pwdLastSet", "msDS-UserPasswordExpiryTimeComputed"
        ]
        self.conn.search(self.base_dn, f"(&(objectClass=user)(sAMAccountName={account_name}))", attributes=attrs)
        if not self.conn.entries:
            return None

        entry = self.conn.entries[0]
        now = datetime.now(timezone.utc)
        uac = int(entry.userAccountControl.value or 0)
        is_never_expire = bool(uac & 0x10000)
        pwd_last_set = entry.pwdLastSet.value
        is_must_change = pwd_last_set in (0, None) or str(pwd_last_set).startswith("1601")

        raw_expiry = getattr(entry, "msDS-UserPasswordExpiryTimeComputed", None)
        expiry_dt = None if (is_must_change or is_never_expire) else self.filetime_to_datetime(raw_expiry.value if raw_expiry else None)
        days = int(round((expiry_dt - now).total_seconds() / 86400)) if expiry_dt else None
        expiry_str = expiry_dt.strftime("%Y-%m-%d %H:%M:%S UTC") if expiry_dt else None

        return {
            "username": str(entry.sAMAccountName or ""),
            "display_name": str(entry.displayName or entry.sAMAccountName or ""),
            "mail": str(entry.mail or ""),
            "days": days,
            "days_remaining": days,
            "expiry_str": expiry_str,
            "pwd_expiry": expiry_str,
            "is_must_change": is_must_change,
            "is_never_expire": is_never_expire,
        }

    def scan_expiring_users(self, warn_days: int = 15, grace_days: int = 7, max_expired_days: int = None, exclude_ou: str = "bots"):
        """扫描域内需要处理的用户（待首次改密、即将到期、宽限期内已逾期）"""
        if max_expired_days is not None:
            grace_days = max_expired_days
        ldap_filter = (
            "(&(objectClass=user)"
            "(!(sAMAccountName=Administrator))"
            "(userAccountControl:1.2.840.113556.1.4.803:=512)"
            "(!(userAccountControl:1.2.840.113556.1.4.803:=2)))"
        )
        attrs = [
            "distinguishedName", "sAMAccountName", "displayName", "mail",
            "userAccountControl", "pwdLastSet", "msDS-UserPasswordExpiryTimeComputed"
        ]
        self.conn.search(search_base=self.base_dn, search_filter=ldap_filter, attributes=attrs)

        now = datetime.now(timezone.utc)
        results = []

        for entry in self.conn.entries:
            dn = str(entry.distinguishedName)
            if exclude_ou and f"OU={exclude_ou}," in dn:
                continue

            uac = int(entry.userAccountControl.value or 0)
            if uac & 0x10000:  # 密码永不过期
                continue

            username = str(entry.sAMAccountName or "")
            display_name = str(entry.displayName or username)
            mail = str(entry.mail or "")

            # 首次改密判断 (新账号或密码被管理员重置)
            pwd_last_set = entry.pwdLastSet.value
            if pwd_last_set in (0, None) or str(pwd_last_set).startswith("1601"):
                results.append({
                    "username": username,
                    "display_name": display_name,
                    "mail": mail,
                    "days": 0,
                    "days_remaining": 0,
                    "expiry_str": "待首次登录重置",
                    "pwd_expiry": "待首次登录重置",
                    "is_must_change": True,
                })
                continue

            raw_expiry = getattr(entry, "msDS-UserPasswordExpiryTimeComputed", None)
            expiry_dt = self.filetime_to_datetime(raw_expiry.value if raw_expiry else None)
            if not expiry_dt:
                continue

            days = int(round((expiry_dt - now).total_seconds() / 86400))
            # 过滤：仅保留临期 (<= warn_days) 并在逾期追溯期内的账号；若 grace_days 为 0 或 None 则不限制逾期下限
            is_within_grace = True if (not grace_days or grace_days <= 0) else (days >= -grace_days)
            if days <= warn_days and is_within_grace:
                expiry_str = expiry_dt.strftime("%Y-%m-%d %H:%M:%S UTC")
                results.append({
                    "username": username,
                    "display_name": display_name,
                    "mail": mail,
                    "days": days,
                    "days_remaining": days,
                    "expiry_str": expiry_str,
                    "pwd_expiry": expiry_str,
                    "is_must_change": False,
                })

        # 待改密排最前，天数越小越靠前
        results.sort(key=lambda x: (1, x["days"]) if not x["is_must_change"] else (0, 0))
        return results
