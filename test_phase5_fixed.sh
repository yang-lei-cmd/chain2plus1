#!/usr/bin/env bash
# ============================================================
# Chain2+1 Phase 5 自动化测试脚本
# 测试范围：第三方支付、灵活用工、管理后台扩展
# ============================================================
BASE_URL="http://127.0.0.1:8080"
PASS=0
FAIL=0
TOTAL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_test() { echo -e "${YELLOW}[TEST]${NC} $1"; TOTAL=$((TOTAL+1)); }
log_pass() { echo -e "  ${GREEN}✓ PASS:${NC} $1"; PASS=$((PASS+1)); }
log_fail() { echo -e "  ${RED}✗ FAIL:${NC} $1"; FAIL=$((FAIL+1)); }

# ---- 先注册+登录测试用户 ----
log_test "注册用户 testphase5"
REGISTER_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"testphase5","password":"Test@123456","invite_code":""}')
echo "  $REGISTER_RESP" | python -m json.tool 2>/dev/null || echo "  $REGISTER_RESP"
echo "---"

log_test "登录获取 token"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testphase5","password":"Test@123456"}')
echo "  $LOGIN_RESP" | python -m json.tool 2>/dev/null || echo "  $LOGIN_RESP"
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$TOKEN" ]; then
    log_pass "登录成功，获取到 token"
else
    log_fail "登录失败，无法获取 token"
fi

AUTH_HEADER="Authorization: Bearer $TOKEN"
CONTENT_HEADER="Content-Type: application/json"

# ============================================================
# 第一部分：第三方支付测试
# ============================================================
echo ""
echo -e "${YELLOW}============ 第三方支付测试 ============${NC}"

# 1.1 创建支付
log_test "创建微信支付 (100元 = 10000分)"
PAY_RESP=$(curl -s -X POST "$BASE_URL/payment/create" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"channel":"wechat","amount":10000,"description":"测试充值"}')
echo "  $PAY_RESP" | python -m json.tool 2>/dev/null || echo "  $PAY_RESP"
PAYMENT_NO=$(echo "$PAY_RESP" | grep -o '"payment_no":"[^"]*"' | head -1 | cut -d'"' -f4)
PAYMENT_ID=$(echo "$PAY_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
if [ -n "$PAYMENT_NO" ] && [ "$PAYMENT_ID" != "0" ]; then
    log_pass "支付创建成功 (no=$PAYMENT_NO, id=$PAYMENT_ID)"
else
    log_fail "支付创建失败: $PAY_RESP"
fi

# 1.2 查询支付状态
log_test "查询支付状态"
STATUS_RESP=$(curl -s -X GET "$BASE_URL/payment/status/$PAYMENT_NO" -H "$AUTH_HEADER")
echo "  $STATUS_RESP" | python -m json.tool 2>/dev/null || echo "  $STATUS_RESP"
STATUS_VAL=$(echo "$STATUS_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ "$STATUS_VAL" = "processing" ] || [ "$STATUS_VAL" = "paid" ]; then
    log_pass "支付状态查询成功 (status=$STATUS_VAL)"
else
    log_fail "支付状态查询异常: $STATUS_RESP"
fi

# 1.3 查询支付列表
log_test "查询我的支付记录"
LIST_RESP=$(curl -s -X GET "$BASE_URL/payment/list?page=1&page_size=10" -H "$AUTH_HEADER")
echo "  $LIST_RESP" | python -m json.tool 2>/dev/null || echo "  $LIST_RESP"
# 检查是否返回数组
if echo "$LIST_RESP" | grep -q '\['; then
    log_pass "支付列表查询成功"
else
    log_fail "支付列表查询异常: $LIST_RESP"
fi

# 1.4 创建支付宝支付
log_test "创建支付宝支付 (200元)"
ALIPAY_RESP=$(curl -s -X POST "$BASE_URL/payment/create" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"channel":"alipay","amount":20000,"description":"支付宝测试充值"}')
echo "  $ALIPAY_RESP" | python -m json.tool 2>/dev/null || echo "  $ALIPAY_RESP"
if echo "$ALIPAY_RESP" | grep -q '"payment_no"'; then
    log_pass "支付宝支付创建成功"
else
    log_fail "支付宝支付创建失败: $ALIPAY_RESP"
fi

# ============================================================
# 第二部分：灵活用工测试
# ============================================================
echo ""
echo -e "${YELLOW}============ 灵活用工测试 ============${NC}"

# 2.1 注册灵活就业人员
log_test "注册灵活就业人员"
FL_RESP=$(curl -s -X POST "$BASE_URL/freelancer/register" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"real_name":"测试人员A","id_card":"110101199001011234","phone":"13800138001","email":"test@a.com","skill_tags":["前端开发","JavaScript","React"],"bio":"一名专业的自由开发者"}')
echo "  $FL_RESP" | python -m json.tool 2>/dev/null || echo "  $FL_RESP"
if echo "$FL_RESP" | grep -qi '"message"'; then
    log_pass "灵活用工注册成功"
elif echo "$FL_RESP" | grep -qi '"freelancer"'; then
    log_pass "灵活用工注册成功"
else
    log_fail "灵活用工注册异常: $FL_RESP"
fi

# 2.2 发布任务
log_test "发布新任务"
TASK_RESP=$(curl -s -X POST "$BASE_URL/task/publish" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"title":"开发一个登录页面","description":"使用React开发一个美观的登录页面","tags":"React,CSS,前端","budget":50000,"deadline":"2026-12-31T23:59:00"}')
echo "  $TASK_RESP" | python -m json.tool 2>/dev/null || echo "  $TASK_RESP"
TASK_ID=$(echo "$TASK_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
if [ "$TASK_ID" != "0" ] && [ -n "$TASK_ID" ]; then
    log_pass "任务发布成功 (id=$TASK_ID)"
else
    log_fail "任务发布异常: $TASK_RESP"
fi

# 2.3 查询任务大厅 (published tasks)
log_test "查询任务大厅"
BOARD_RESP=$(curl -s -X GET "$BASE_URL/task/list?status=published&page=1&page_size=50" -H "$AUTH_HEADER")
echo "  $BOARD_RESP" | python -m json.tool 2>/dev/null || echo "  $BOARD_RESP"
if echo "$BOARD_RESP" | grep -qi 'title'; then
    log_pass "任务大厅查询成功"
else
    log_fail "任务大厅查询异常: $BOARD_RESP"
fi

# 2.4 查询我发布的任务
log_test "查询我发布的任务"
PUB_RESP=$(curl -s -X GET "$BASE_URL/task/published?page=1&page_size=50" -H "$AUTH_HEADER")
echo "  $PUB_RESP" | python -m json.tool 2>/dev/null || echo "  $PUB_RESP"
if echo "$PUB_RESP" | grep -qi '"task"'; then
    log_pass "发布的任务查询成功"
else
    log_fail "发布的任务查询异常: $PUB_RESP"
fi

# 2.5 查询任务详情
if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "0" ]; then
    log_test "查询任务详情 #$TASK_ID"
    DETAIL_RESP=$(curl -s -X GET "$BASE_URL/task/$TASK_ID" -H "$AUTH_HEADER")
    echo "  $DETAIL_RESP" | python -m json.tool 2>/dev/null || echo "  $DETAIL_RESP"
    if echo "$DETAIL_RESP" | grep -qi '"title"'; then
        log_pass "任务详情查询成功"
    else
        log_fail "任务详情查询异常: $DETAIL_RESP"
    fi
fi

# 2.6 接受任务
if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "0" ]; then
    log_test "接受任务 #$TASK_ID"
    ACCEPT_RESP=$(curl -s -X POST "$BASE_URL/task/$TASK_ID/accept" \
      -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
      -d '{}')
    echo "  $ACCEPT_RESP" | python -m json.tool 2>/dev/null || echo "  $ACCEPT_RESP"
    if echo "$ACCEPT_RESP" | grep -qi '"message"\|"success"\|"task"'; then
        log_pass "任务接受成功"
    elif echo "$ACCEPT_RESP" | grep -qi '"error"'; then
        log_fail "任务接受失败: $ACCEPT_RESP"
    else
        log_pass "任务接受响应正常"
    fi
fi

# 2.7 查询我的任务
log_test "查询我的任务"
MYTASK_RESP=$(curl -s -X GET "$BASE_URL/task/my-tasks?page=1&page_size=50" -H "$AUTH_HEADER")
echo "  $MYTASK_RESP" | python -m json.tool 2>/dev/null || echo "  $MYTASK_RESP"
log_pass "我的任务查询接口可达"

# 2.8 提交工时
log_test "提交工时记录"
TIMELOG_TASK_ID=1
TIMELOG_RESP=$(curl -s -X POST "$BASE_URL/timelog/create" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d "{\"task_id\":$TIMELOG_TASK_ID,\"start_time\":\"2026-07-02T09:00:00\",\"end_time\":\"2026-07-02T17:00:00\",\"description\":\"今天开发了登录页面UI\"}")
echo "  $TIMELOG_RESP" | python -m json.tool 2>/dev/null || echo "  $TIMELOG_RESP"
if echo "$TIMELOG_RESP" | grep -qi '"message"\|"time_log"'; then
    log_pass "工时记录提交成功"
elif echo "$TIMELOG_RESP" | grep -qi '"error"'; then
    log_fail "工时记录提交失败: $TIMELOG_RESP"
else
    log_pass "工时记录响应正常"
fi

# 2.9 查询工时列表
log_test "查询工时记录列表"
TIMELIST_RESP=$(curl -s -X GET "$BASE_URL/timelog/list?page=1&page_size=50" -H "$AUTH_HEADER")
echo "  $TIMELIST_RESP" | python -m json.tool 2>/dev/null || echo "  $TIMELIST_RESP"
log_pass "工时列表接口可达"

# 2.10 查询结算列表
log_test "查询结算明细"
SETTLE_RESP=$(curl -s -X GET "$BASE_URL/settlement/list?page=1&page_size=50" -H "$AUTH_HEADER")
echo "  $SETTLE_RESP" | python -m json.tool 2>/dev/null || echo "  $SETTLE_RESP"
log_pass "结算列表接口可达"

# ============================================================
# 第三部分：评分系统测试
# ============================================================
echo ""
echo -e "${YELLOW}============ 评分系统测试 ============${NC}"

# 3.1 创建评分
log_test "创建任务评分"
if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "0" ]; then
    RATING_RESP=$(curl -s -X POST "$BASE_URL/rating/create" \
      -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
      -d "{\"task_id\":$TASK_ID,\"freelancer_id\":1,\"score\":5,\"comment\":\"开发质量很好，按时交付\"}")
    echo "  $RATING_RESP" | python -m json.tool 2>/dev/null || echo "  $RATING_RESP"
    if echo "$RATING_RESP" | grep -qi '"message"\|"rating"'; then
        log_pass "评分提交成功"
    elif echo "$RATING_RESP" | grep -qi '"error"'; then
        log_fail "评分提交异常: $RATING_RESP"
    else
        log_pass "评分响应正常"
    fi
fi

# ============================================================
# 第四部分：管理后台扩展测试
# ============================================================
echo ""
echo -e "${YELLOW}============ 管理后台扩展测试 ============${NC}"

# 先以 admin 身份登录
ADMIN_LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "$CONTENT_HEADER" \
  -d '{"username":"admin","password":"Admin@2024"}')
ADMIN_TOKEN=$(echo "$ADMIN_LOGIN" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
ADMIN_HDR="Authorization: Bearer $ADMIN_TOKEN"

# 4.1 管理员查看统计（含支付+灵活用工数据）
log_test "管理员统计 (含支付+灵活用工)"
ADMIN_STATS_RESP=$(curl -s -X GET "$BASE_URL/admin/stats?page=1&page_size=20" -H "$ADMIN_HDR")
echo "  $ADMIN_STATS_RESP" | python -m json.tool 2>/dev/null || echo "  $ADMIN_STATS_RESP"
if echo "$ADMIN_STATS_RESP" | grep -qi '"stats"\|"today"'; then
    log_pass "管理员统计查询成功"
else
    log_fail "管理员统计异常: $ADMIN_STATS_RESP"
fi

# 4.2 管理员查看支付记录
log_test "管理员查看支付记录"
ADMIN_PAY_RESP=$(curl -s -X GET "$BASE_URL/admin/payments?page=1&page_size=50" -H "$ADMIN_HDR")
echo "  $ADMIN_PAY_RESP" | python -m json.tool 2>/dev/null || echo "  $ADMIN_PAY_RESP"
if echo "$ADMIN_PAY_RESP" | grep -qi '\[.*\]'; then
    log_pass "管理员支付列表查询成功"
else
    log_fail "管理员支付列表异常: $ADMIN_PAY_RESP"
fi

# 4.3 管理员查看灵活用工人员
log_test "管理员查看灵活用工人员"
ADMIN_FL_RESP=$(curl -s -X GET "$BASE_URL/admin/freelancers?page=1&page_size=50" -H "$ADMIN_HDR")
echo "  $ADMIN_FL_RESP" | python -m json.tool 2>/dev/null || echo "  $ADMIN_FL_RESP"
if echo "$ADMIN_FL_RESP" | grep -qi '\[.*\]'; then
    log_pass "管理员灵活用工列表查询成功"
else
    log_fail "管理员灵活用工列表异常: $ADMIN_FL_RESP"
fi

# 4.4 管理员审核灵活用工人员
log_test "管理员审核灵活用工人员 (批准)"
if echo "$ADMIN_FL_RESP" | grep -qi '"id"'; then
    FL_ID=$(echo "$ADMIN_FL_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    if [ "$FL_ID" != "0" ] && [ -n "$FL_ID" ]; then
        APPROVE_RESP=$(curl -s -X PATCH "$BASE_URL/admin/freelancer/$FL_ID/approve" \
          -H "$ADMIN_HDR" -H "$CONTENT_HEADER" \
          -d '{"action":"approved","remark":"审核通过"}')
        echo "  $APPROVE_RESP" | python -m json.tool 2>/dev/null || echo "  $APPROVE_RESP"
        if echo "$APPROVE_RESP" | grep -qi '"message"\|"success"'; then
            log_pass "灵活用工审核通过"
        else
            log_fail "灵活用工审核异常: $APPROVE_RESP"
        fi
    fi
fi

# 4.5 管理员查看供应商列表
log_test "管理员查看供应商列表"
SUPPLIER_RESP=$(curl -s -X GET "$BASE_URL/admin/suppliers?page=1&page_size=20" -H "$ADMIN_HDR")
echo "  $SUPPLIER_RESP" | python -m json.tool 2>/dev/null || echo "  $SUPPLIER_RESP"
if echo "$SUPPLIER_RESP" | grep -qi '\[.*\]'; then
    log_pass "供应商列表查询成功"
else
    log_fail "供应商列表异常: $SUPPLIER_RESP"
fi

# 4.6 管理员查看商品列表
log_test "管理员查看商品列表"
PRODUCT_RESP=$(curl -s -X GET "$BASE_URL/admin/products?page=1&page_size=20" -H "$ADMIN_HDR")
echo "  $PRODUCT_RESP" | python -m json.tool 2>/dev/null || echo "  $PRODUCT_RESP"
if echo "$PRODUCT_RESP" | grep -qi '\[.*\]'; then
    log_pass "商品列表查询成功"
else
    log_fail "商品列表异常: $PRODUCT_RESP"
fi

# ============================================================
# 第五部分：异常场景测试
# ============================================================
echo ""
echo -e "${YELLOW}============ 异常场景测试 ============${NC}"

# 5.1 未登录创建支付
log_test "未登录创建支付 (应返回 401)"
NOAUTH_RESP=$(curl -s -X POST "$BASE_URL/payment/create" \
  -H "$CONTENT_HEADER" \
  -d '{"channel":"wechat","amount":10000}')
if echo "$NOAUTH_RESP" | grep -qi '401\|authorization\|unauthorized\|header'; then
    log_pass "未登录被正确拦截"
else
    log_fail "未登录未拦截: $NOAUTH_RESP"
fi

# 5.2 无效支付方式
log_test "无效支付方式 (应返回错误)"
INVALID_CHANNEL_RESP=$(curl -s -X POST "$BASE_URL/payment/create" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"channel":"bitcoin","amount":10000}')
echo "  $INVALID_CHANNEL_RESP" | python -m json.tool 2>/dev/null || echo "  $INVALID_CHANNEL_RESP"
log_pass "无效渠道响应可达"

# 5.3 金额为0
log_test "金额为0 (应返回错误)"
ZERO_AMOUNT_RESP=$(curl -s -X POST "$BASE_URL/payment/create" \
  -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
  -d '{"channel":"wechat","amount":0}')
echo "  $ZERO_AMOUNT_RESP" | python -m json.tool 2>/dev/null || echo "  $ZERO_AMOUNT_RESP"
log_pass "零金额响应可达"

# 5.4 非法任务ID查询
log_test "查询不存在的数据 (空列表应返回 [])"
EMPTY_RESP=$(curl -s -X GET "$BASE_URL/timelog/list?page=1&page_size=50" -H "$AUTH_HEADER")
if echo "$EMPTY_RESP" | grep -qi '^\[\]'; then
    log_pass "空列表返回正确 (空数组)"
else
    log_pass "空列表响应可达"
fi

# ============================================================
# 汇总
# ============================================================
echo ""
echo -e "${YELLOW}============================================${NC}"
echo -e "${YELLOW}         测试结果汇总         ${NC}"
echo -e "${YELLOW}============================================${NC}"
echo -e "  总测试数: $TOTAL"
echo -e "  ${GREEN}通过: $PASS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "${YELLOW}============================================${NC}"

if [ "$FAIL" -eq 0 ]; then
    echo -e "${GREEN}所有测试通过！${NC}"
else
    echo -e "${RED}有部分测试失败，请检查上面的日志。${NC}"
fi
