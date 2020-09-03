import React, {useState} from "react";
import {Box, HStack, IconButton, Select, Skeleton, useColorModeValue, VStack,} from "@chakra-ui/core";
import {FaAngleDoubleLeft, FaAngleDoubleRight, FaAngleLeft, FaAngleRight,} from "react-icons/fa";
import {User} from "src/types";

import classNames from "classnames/bind";
import styles from "./Table.module.scss";

const cx = classNames.bind(styles);

export interface Props {
  isLoaded: boolean;
  users: Array<User>;
  limit: number;
  page: number;
  total: number;
  onPageChange?: (v: number) => void;
  onLimitChange?: (v: number) => void;
}

const UserTable = (props: Props) => {
  const [page, setPage] = useState(props.page);
  const [limit, setLimit] = useState(props.limit);

  const tableMode = useColorModeValue("table-light", "table-dark");

  const tableStyles = cx({
    table: true,
    "table-striped": true,
    "table-dark": tableMode === "table-dark",
    "table-light": tableMode === "table-light",
  });

  const handlePageChange = (newPage: number) => {
    if (newPage < 1) {
      newPage = 1;
    }
    if (newPage > Math.ceil(props.total / limit)) {
      newPage = 1;
    }
    setPage(newPage);
    props.onPageChange && props.onPageChange(newPage);
  };

  const handleLimitChange = (newLimit: number) => {
    setLimit(newLimit);
    props.onLimitChange && props.onLimitChange(newLimit);
  };

  return (
    <VStack d="flex" alignContent="center">
      <Box as="table" className={tableStyles}>
        <Box as="thead">
          <Box as="tr">
            <Box as="th" scope="col">
              #
            </Box>
            <Box as="th" scope="col">
              Email
            </Box>
            <Box as="th" scope="col">
              ID
            </Box>
          </Box>
        </Box>
        <Box as="tbody">
          {!props.isLoaded &&
          [...Array(limit)].map((x, i) => (
            <Box as="tr" key={i}>
              <Box as="th" scope="row">
                <Skeleton height="30px"/>
              </Box>
              <Box as="th">
                <Skeleton height="30px"/>
              </Box>
              <Box as="th">
                <Skeleton height="30px"/>
              </Box>
            </Box>
          ))}
          {props.isLoaded &&
          props.users.map((user, idx) => (
            <Box as="tr" key={user.id}>
              <Box as="th" scope="row">
                {idx}
              </Box>
              <Box as="th">{user.email}</Box>
              <Box as="th">{user.id}</Box>
            </Box>
          ))}
        </Box>
      </Box>
      <HStack justifyContent="space-around" alignItems="baseline">
        <IconButton
          aria-label=""
          mx={1}
          onClick={() => handlePageChange(0)}
          icon={<FaAngleDoubleLeft/>}
        />
        <IconButton
          aria-label=""
          mx={1}
          onClick={() => handlePageChange(page - 1)}
          icon={<FaAngleLeft/>}
        />
        <Select
          mx={1}
          variant="unstyled"
          onChange={(e) => handleLimitChange(Number(e.target.value))}
          value={limit}
        >
          <option aria-label="" value={10}>
            10
          </option>
          <option aria-label="" value={100}>
            100
          </option>
        </Select>
        <IconButton
          aria-label=""
          mx={1}
          onClick={() => handlePageChange(page + 1)}
          icon={<FaAngleRight/>}
        />
        <IconButton
          aria-label=""
          mx={1}
          onClick={() => handlePageChange(Math.ceil(props.total / page))}
          icon={<FaAngleDoubleRight/>}
        />
      </HStack>
    </VStack>
  );
};

export default UserTable;
